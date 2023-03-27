// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package dbaccessor

import (
	"context"
	"database/sql"
	"net"
	"sync"
	"time"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/worker/v3"
	"github.com/juju/worker/v3/catacomb"
	"github.com/juju/worker/v3/dependency"
	"gopkg.in/tomb.v2"

	coredatabase "github.com/juju/juju/core/database"
	"github.com/juju/juju/database"
	"github.com/juju/juju/database/app"
	"github.com/juju/juju/database/dqlite"
	"github.com/juju/juju/pubsub/apiserver"
)

const (
	// PollInterval is the amount of time to wait between polling the database.
	PollInterval = time.Second * 10

	// DefaultVerifyAttempts is the number of attempts to verify the database,
	// by opening a new database on verification failure.
	DefaultVerifyAttempts = 3
)

// TrackedDB defines the union of a TrackedDB and a worker.Worker interface.
// This is local to the package, allowing for better testing of the underlying
// trackerDB worker.
type TrackedDB interface {
	coredatabase.TrackedDB
	worker.Worker
}

// NodeManager creates Dqlite `App` initialisation arguments and options.
type NodeManager interface {
	// IsExistingNode returns true if this machine of container has run a
	// Dqlite node in the past.
	IsExistingNode() (bool, error)

	// IsBootstrappedNode returns true if this machine or container was where
	// we first bootstrapped Dqlite, and it hasn't been reconfigured since.
	IsBootstrappedNode(context.Context) (bool, error)

	// EnsureDataDir ensures that a directory for Dqlite data exists at
	// a path determined by the agent config, then returns that path.
	EnsureDataDir() (string, error)

	// ClusterServers returns the node information for
	// Dqlite nodes configured to be in the cluster.
	ClusterServers(context.Context) ([]dqlite.NodeInfo, error)

	//SetClusterServers reconfigures the Dqlite cluster members.
	SetClusterServers(context.Context, []dqlite.NodeInfo) error

	// SetNodeInfo rewrites the local node information
	// file in the Dqlite data directory.
	SetNodeInfo(dqlite.NodeInfo) error

	// WithLogFuncOption returns a Dqlite application Option that will proxy Dqlite
	// log output via this factory's logger where the level is recognised.
	WithLogFuncOption() app.Option

	// WithAddressOption returns a Dqlite application Option
	// for specifying the local address:port to use.
	WithAddressOption() (app.Option, error)

	// WithTLSOption returns a Dqlite application Option for TLS encryption
	// of traffic between clients and clustered application nodes.
	WithTLSOption() (app.Option, error)

	// WithClusterOption returns a Dqlite application Option for initialising
	// Dqlite as the member of a cluster with peers representing other controllers.
	WithClusterOption() (app.Option, error)
}

// DBGetter describes the ability to supply a sql.DB
// reference for a particular database.
type DBGetter interface {
	// GetDB returns a sql.DB reference for the dqlite-backed database that
	// contains the data for the specified namespace.
	// A NotFound error is returned if the worker is unaware of the requested DB.
	GetDB(namespace string) (coredatabase.TrackedDB, error)
}

// DBApp describes methods of a Dqlite database application,
// required to run this host as a Dqlite node.
type DBApp interface {
	// Open the dqlite database with the given name
	Open(context.Context, string) (*sql.DB, error)

	// Ready can be used to wait for a node to complete tasks that
	// are initiated at startup. For example a new node will attempt
	// to join the cluster, a restarted node will check if it should
	// assume some particular role, etc.
	//
	// If this method returns without error it means that those initial
	// tasks have succeeded and follow-up operations like Open() are more
	// likely to succeed quickly.
	Ready(context.Context) error

	// Handover transfers all responsibilities for this node (such has
	// leadership and voting rights) to another node, if one is available.
	//
	// This method should always be called before invoking Close(),
	// in order to gracefully shut down a node.
	Handover(context.Context) error

	// ID returns the dqlite ID of this application node.
	ID() uint64

	// Close the application node, releasing all resources it created.
	Close() error
}

// dbRequest is used to pass requests for TrackedDB
// instances into the worker loop.
type dbRequest struct {
	namespace string
	done      chan struct{}
}

// makeDBRequest creates a new TrackedDB request for the input namespace.
func makeDBRequest(namespace string) dbRequest {
	return dbRequest{
		namespace: namespace,
		done:      make(chan struct{}),
	}
}

// WorkerConfig encapsulates the configuration options for the
// dbaccessor worker.
type WorkerConfig struct {
	NodeManager NodeManager
	Clock       clock.Clock

	// Hub is the pub/sub central hub used to receive notifications
	// about API server topology changes.
	Hub         Hub
	Logger      Logger
	NewApp      func(string, ...app.Option) (DBApp, error)
	NewDBWorker func(DBApp, string, ...TrackedDBWorkerOption) (TrackedDB, error)

	// ControllerID uniquely identifies the controller that this
	// worker is running on. It is equivalent to the machine ID.
	ControllerID string
}

// Validate ensures that the config values are valid.
func (c *WorkerConfig) Validate() error {
	if c.NodeManager == nil {
		return errors.NotValidf("missing NodeManager")
	}
	if c.Clock == nil {
		return errors.NotValidf("missing Clock")
	}
	if c.Hub == nil {
		return errors.NotValidf("missing Hub")
	}
	if c.Logger == nil {
		return errors.NotValidf("missing Logger")
	}
	if c.NewApp == nil {
		return errors.NotValidf("missing NewApp")
	}
	if c.NewDBWorker == nil {
		return errors.NotValidf("missing NewDBWorker")
	}
	return nil
}

type dbWorker struct {
	cfg      WorkerConfig
	catacomb catacomb.Catacomb

	mu       sync.RWMutex
	dbApp    DBApp
	dbRunner *worker.Runner

	// dbRequests is used to synchronise GetDB
	// requests into this worker's event loop.
	dbRequests chan dbRequest

	// apiServerChanges is used to handle incoming changes
	// to API server details within the worker loop.
	apiServerChanges chan apiserver.Details
}

func newWorker(cfg WorkerConfig) (*dbWorker, error) {
	var err error
	if err = cfg.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	w := &dbWorker{
		cfg: cfg,
		dbRunner: worker.NewRunner(worker.RunnerParams{
			Clock: cfg.Clock,
			// If a worker goes down, we've attempted multiple retries and in
			// that case we do want to cause the dbaccessor to go down. This
			// will then bring up a new dqlite app.
			IsFatal:      func(err error) bool { return true },
			RestartDelay: time.Second * 30,
		}),
		dbRequests:       make(chan dbRequest),
		apiServerChanges: make(chan apiserver.Details),
	}

	if err = catacomb.Invoke(catacomb.Plan{
		Site: &w.catacomb,
		Work: w.loop,
		Init: []worker.Worker{
			w.dbRunner,
		},
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return w, nil
}

func (w *dbWorker) loop() (err error) {
	defer w.shutdownDqlite(context.TODO())

	extant, err := w.cfg.NodeManager.IsExistingNode()
	if err != nil {
		return errors.Trace(err)
	}

	// At this time, while Juju is using both Mongo and Dqlite, we piggyback
	// off the peer-grouper, which applies any configured HA space and
	// broadcasts clustering addresses. Once we do away with mongo,
	// that worker will be replaced with a Dqlite-focussed analogue that does
	// largely the same thing, though potentially disseminating changes via a
	// mechanism other than pub/sub.
	unsub, err := w.cfg.Hub.Subscribe(apiserver.DetailsTopic, w.handleAPIServerChangeMsg)
	if err != nil {
		return errors.Annotate(err, "subscribing to API server topology changes")
	}
	defer unsub()

	// If this is an existing node, we start it up immediately.
	// Otherwise, this host is entering a HA cluster, and we need to wait for
	// the peer-grouper to determine and broadcast addresses satisfying the
	// Juju HA space (if configured); request those details.
	// Once received we can continue configuring this node as a member.
	if extant {
		if err := w.startExistingDqliteNode(); err != nil {
			return errors.Trace(err)
		}
	} else {
		if _, err := w.cfg.Hub.Publish(apiserver.DetailsRequestTopic, apiserver.DetailsRequest{
			Requester: "db-accessor",
			LocalOnly: true,
		}); err != nil {
			return errors.Trace(err)
		}
	}

	for {
		select {
		case req := <-w.dbRequests:
			if err := w.openDatabase(req.namespace); err != nil {
				w.cfg.Logger.Errorf("opening database %q: %s", req.namespace, err.Error())
			}
			close(req.done)
		case <-w.catacomb.Dying():
			return w.catacomb.ErrDying()
		case apiDetails := <-w.apiServerChanges:
			if err := w.processAPIServerChange(apiDetails); err != nil {
				return errors.Trace(err)
			}
		}
	}
}

// Kill is part of the worker.Worker interface.
func (w *dbWorker) Kill() {
	w.catacomb.Kill(nil)
}

// Wait is part of the worker.Worker interface.
func (w *dbWorker) Wait() error {
	return w.catacomb.Wait()
}

// GetDB returns a TrackedDB reference for the dqlite-backed
// database that contains the data for the specified namespace.
// TODO (stickupkid): Before handing out any DB for any namespace,
// we should first validate it exists in the controller list.
// This should only be required if it's not the controller DB.
func (w *dbWorker) GetDB(namespace string) (coredatabase.TrackedDB, error) {
	// Enqueue the request.
	req := makeDBRequest(namespace)
	select {
	case w.dbRequests <- req:
	case <-w.catacomb.Dying():
		return nil, w.catacomb.ErrDying()
	}

	// Wait for the worker loop to indicate it's done.
	select {
	case <-req.done:
	case <-w.catacomb.Dying():
		return nil, w.catacomb.ErrDying()
	}

	// This will return a not found error if the request was not honoured.
	// The error will be logged - we don't crash this worker for bad calls.
	tracked, err := w.dbRunner.Worker(namespace, w.catacomb.Dying())
	if err != nil {
		return nil, errors.Trace(err)
	}
	return tracked.(coredatabase.TrackedDB), nil
}

// startExistingDqliteNode takes care of starting Dqlite
// when this host has run a node previously.
func (w *dbWorker) startExistingDqliteNode() error {
	w.cfg.Logger.Infof("host is configured as a Dqlite node")

	mgr := w.cfg.NodeManager

	asBootstrapped, err := mgr.IsBootstrappedNode(context.TODO())
	if err != nil {
		return errors.Trace(err)
	}

	// If this existing node is not as bootstrapped, then it is part of a
	// cluster. The Dqlite Raft log and configuration in the Dqlite data
	// directory will indicate the cluster members, but we need to ensure
	// TLS for traffic between nodes explicitly.
	var options []app.Option
	if !asBootstrapped {
		withTLS, err := mgr.WithTLSOption()
		if err != nil {
			return errors.Trace(err)
		}
		options = append(options, withTLS)
	}

	return errors.Trace(w.initialiseDqlite(options...))
}

func (w *dbWorker) initialiseDqlite(options ...app.Option) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.dbApp != nil {
		return nil
	}

	mgr := w.cfg.NodeManager

	dataDir, err := mgr.EnsureDataDir()
	if err != nil {
		return errors.Trace(err)
	}

	if w.dbApp, err = w.cfg.NewApp(dataDir, append(options, mgr.WithLogFuncOption())...); err != nil {
		return errors.Trace(err)
	}

	ctx := context.TODO()

	if err := w.dbApp.Ready(ctx); err != nil {
		return errors.Annotatef(err, "ensuring Dqlite is ready to process changes")
	}

	// Open up the default controller database. Other database namespaces can
	// be opened up in a more lazy fashion.
	if err := w.openDatabase(coredatabase.ControllerNS); err != nil {
		return errors.Annotate(err, "opening initial databases")
	}

	w.cfg.Logger.Infof("initialized Dqlite application (ID: %v)", w.dbApp.ID())
	return nil
}

func (w *dbWorker) openDatabase(namespace string) error {
	err := w.dbRunner.StartWorker(namespace, func() (worker.Worker, error) {
		return w.cfg.NewDBWorker(w.dbApp, namespace, WithClock(w.cfg.Clock), WithLogger(w.cfg.Logger))
	})
	if errors.Is(err, errors.AlreadyExists) {
		return nil
	}
	return errors.Trace(err)
}

// TrackedDBWorkerOption is a function that configures a TrackedDBWorker.
type TrackedDBWorkerOption func(*trackedDBWorker)

// WithVerifyDBFunc sets the function used to verify the database connection.
func WithVerifyDBFunc(f func(context.Context, *sql.DB) error) TrackedDBWorkerOption {
	return func(w *trackedDBWorker) {
		w.verifyDBFunc = f
	}
}

// WithClock sets the clock used by the worker.
func WithClock(clock clock.Clock) TrackedDBWorkerOption {
	return func(w *trackedDBWorker) {
		w.clock = clock
	}
}

// WithLogger sets the logger used by the worker.
func WithLogger(logger Logger) TrackedDBWorkerOption {
	return func(w *trackedDBWorker) {
		w.logger = logger
	}
}

type trackedDBWorker struct {
	tomb tomb.Tomb

	dbApp     DBApp
	namespace string

	mutex sync.RWMutex
	db    *sql.DB
	err   error

	clock  clock.Clock
	logger Logger

	verifyDBFunc func(context.Context, *sql.DB) error
}

// NewTrackedDBWorker creates a new TrackedDBWorker
func NewTrackedDBWorker(dbApp DBApp, namespace string, opts ...TrackedDBWorkerOption) (TrackedDB, error) {
	w := &trackedDBWorker{
		dbApp:        dbApp,
		namespace:    namespace,
		clock:        clock.WallClock,
		verifyDBFunc: defaultVerifyDBFunc,
	}

	for _, opt := range opts {
		opt(w)
	}

	var err error
	w.db, err = w.dbApp.Open(context.TODO(), w.namespace)
	if err != nil {
		return nil, errors.Trace(err)
	}

	w.tomb.Go(w.loop)

	return w, nil
}

// DB closes over a raw *sql.DB. Closing over the DB allows the late
// realization of the database. Allowing retries of DB acquisition if there
// is a failure that is non-retryable.
func (w *trackedDBWorker) DB(fn func(*sql.DB) error) error {
	w.mutex.RLock()
	// We have a fatal error, the DB can not be accessed.
	if w.err != nil {
		w.mutex.RUnlock()
		return errors.Trace(w.err)
	}
	db := w.db
	w.mutex.RUnlock()

	return fn(db)
}

// Txn closes over a raw *sql.Tx. This allows retry semantics in only one
// location. For instances where the underlying sql database is busy or if
// it's a common retryable error that can be handled cleanly in one place.
func (w *trackedDBWorker) Txn(ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	return w.DB(func(db *sql.DB) error {
		return database.Retry(ctx, func() error {
			return database.Txn(ctx, db, fn)
		})
	})
}

// Err will return any fatal errors that have occurred on the worker, trying
// to acquire the database.
func (w *trackedDBWorker) Err() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	return w.err
}

// Kill implements worker.Worker
func (w *trackedDBWorker) Kill() {
	w.tomb.Kill(nil)
}

// Wait implements worker.Worker
func (w *trackedDBWorker) Wait() error {
	return w.tomb.Wait()
}

func (w *trackedDBWorker) loop() error {
	timer := w.clock.NewTimer(PollInterval)
	defer timer.Stop()

	for {
		select {
		case <-w.tomb.Dying():
			return tomb.ErrDying
		case <-timer.Chan():
			// Any retryable errors are handled at the txn level. If we get an
			// error returning here, we've either exhausted the number of
			// retries or the error was fatal.
			w.mutex.RLock()
			currentDB := w.db
			w.mutex.RUnlock()

			newDB, err := w.verifyDB(currentDB)
			if err != nil {
				// If we get an error, ensure we close the underlying db and
				// mark the tracked db in an error state.
				w.mutex.Lock()
				if err := w.db.Close(); err != nil {
					w.logger.Errorf("error closing database: %v", err)
				}
				w.err = errors.Trace(err)
				w.mutex.Unlock()

				// As we failed attempting to verify the db, we're in a fatal
				// state. Collapse the worker and if required, cause the other
				// workers to follow suite.
				return errors.Trace(err)
			}

			// We've got a new DB. Close the old one and replace it with the
			// new one, if they're not the same.
			if newDB != currentDB {
				w.mutex.Lock()
				if err := w.db.Close(); err != nil {
					w.logger.Errorf("error closing database: %v", err)
				}
				w.db = newDB
				w.err = nil
				w.mutex.Unlock()
			}
		}
	}
}

func (w *trackedDBWorker) verifyDB(db *sql.DB) (*sql.DB, error) {
	// Force the timeout to be lower that the DefaultTimeout,
	// so we can spot issues sooner.
	// Also allow killing the tomb to cancel the context,
	// so shutdown/restart can not be blocked by this call.
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	ctx = w.tomb.Context(ctx)
	defer cancel()

	if w.logger.IsTraceEnabled() {
		w.logger.Tracef("verifying database by attempting a ping")
	}

	// There are multiple levels of retries here. For good reason. We want to
	// retry the transaction semantics for retryable errors. Then if that fails
	// we want to retry if the database is at fault. In that case we want
	// to open up a new database and try the transaction again.
	for i := 0; i < DefaultVerifyAttempts; i++ {
		// Verify that we don't have a potential nil database from the retry
		// semantics.
		if db == nil {
			return nil, errors.NotFoundf("database")
		}

		err := database.Retry(ctx, func() error {
			if w.logger.IsTraceEnabled() {
				w.logger.Tracef("attempting ping")
			}
			return w.verifyDBFunc(ctx, db)
		})
		// We were successful at requesting the schema, so we can bail out
		// early.
		if err == nil {
			return db, nil
		}

		// We failed to apply the transaction, so just return the error and
		// cause the worker to crash.
		if i == DefaultVerifyAttempts-1 {
			return nil, errors.Trace(err)
		}

		// We got an error that is non-retryable, attempt to open a new database
		// connection and see if that works.
		w.logger.Errorf("unable to ping db: attempting to reopen the database before attempting again: %v", err)

		// Attempt to open a new database. If there is an error, just crash
		// the worker, we can't do anything else.
		if db, err = w.dbApp.Open(ctx, w.namespace); err != nil {
			return nil, errors.Trace(err)
		}
	}
	return nil, errors.NotValidf("database")
}

func defaultVerifyDBFunc(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}

// handleAPIServerChangeMsg is the callback supplied to the pub/sub
// subscription for API server details. It effectively synchronises the
// handling of such messages into the worker's evert loop.
func (w *dbWorker) handleAPIServerChangeMsg(_ string, apiDetails apiserver.Details, err error) {
	if err != nil {
		// This should never happen.
		w.cfg.Logger.Errorf("pub/sub callback error: %v", err)
		return
	}

	select {
	case <-w.catacomb.Dying():
	case w.apiServerChanges <- apiDetails:
	}
}

// processAPIServerChange deals with cluster topology changes.
func (w *dbWorker) processAPIServerChange(apiDetails apiserver.Details) error {
	w.cfg.Logger.Debugf("new API server details: %#v", apiDetails)

	mgr := w.cfg.NodeManager

	extant, err := mgr.IsExistingNode()
	if err != nil {
		return errors.Trace(err)
	}

	ctx := context.TODO()

	// If this is an existing node, check if we are bound to the loopback IP
	// and entering HA, which requires a new local-cloud bind address.
	if extant {
		asBootstrapped, err := mgr.IsBootstrappedNode(ctx)
		if err != nil {
			return errors.Trace(err)
		}

		// If this is a node that is already clustered, there is nothing to do.
		if !asBootstrapped {
			w.cfg.Logger.Debugf("node already clustered; no work to do")
			return nil
		}

		// Look for *our* internal address in the details that were broadcast.
		// This is the same local-cloud address used by Mongo for replication.
		hostPort := apiDetails.Servers[w.cfg.ControllerID].InternalAddress
		if hostPort == "" {
			return errors.New("no internal address determined for this Dqlite node to bind to")
		}
		addr, _, err := net.SplitHostPort(hostPort)
		if err != nil {
			return errors.Annotatef(err, "splitting host/port for %s", hostPort)
		}
		if err := w.rebindAddress(ctx, addr); err != nil {
			return errors.Trace(err)
		}

		// Just bounce wholesale after reconfiguring the node.
		// We flush the opened DBs and ensure a clean slate for dependents.
		w.cfg.Logger.Infof("successfully reconfigured Dqlite; restarting worker")
		return dependency.ErrBounce
	}

	// Otherwise this is a node added by enabling HA,
	// and we need to join to an existing cluster.

	return nil
}

// rebindAddress stops the current node, reconfigures the cluster so that
// it is a single server bound to the input local-cloud address.
// It should be called only for a cluster constituted by a single node
// bound to the loopback IP address.
func (w *dbWorker) rebindAddress(ctx context.Context, addr string) error {
	w.shutdownDqlite(ctx)

	mgr := w.cfg.NodeManager
	servers, err := mgr.ClusterServers(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	// This should be implied by an earlier check of
	// NodeManager.IsBootstrappedNode, but we want to guard very
	// conservatively against breaking established clusters.
	if len(servers) != 1 {
		w.cfg.Logger.Debugf("not a singular server; skipping address rebind")
		return nil
	}

	// We need to preserve the port from the existing address.
	_, port, err := net.SplitHostPort(servers[0].Address)
	if err != nil {
		return errors.Trace(err)
	}
	servers[0].Address = net.JoinHostPort(addr, port)

	w.cfg.Logger.Infof("rebinding Dqlite node to %s", addr)
	if err := mgr.SetClusterServers(ctx, servers); err != nil {
		return errors.Trace(err)
	}

	return errors.Trace(mgr.SetNodeInfo(servers[0]))
}

// shutdownDqlite makes a best-effort attempt to hand off and shut
// down the local Dqlite node.
// We reassign the dbReady channel, which blocks GetDB requests.
// If the worker is not shutting down permanently, Dqlite should be
// reinitialised either directly or by bouncing the agent reasonably
// soon after calling this method.
func (w *dbWorker) shutdownDqlite(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.dbApp == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*30)
	defer cancel()

	if err := w.dbApp.Handover(ctx); err != nil {
		w.cfg.Logger.Errorf("handing off Dqlite responsibilities: %v", err)
	}

	if err := w.dbApp.Close(); err != nil {
		w.cfg.Logger.Errorf("closing Dqlite application: %v", err)
	}

	w.dbApp = nil
}