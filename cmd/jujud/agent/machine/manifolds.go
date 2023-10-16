// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package machine

import (
	stdcontext "context"
	"net/http"
	"runtime"
	"time"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/proxy"
	"github.com/juju/pubsub/v2"
	"github.com/juju/utils/v3/voyeur"
	"github.com/juju/version/v2"
	"github.com/juju/worker/v3"
	"github.com/juju/worker/v3/dependency"
	"github.com/prometheus/client_golang/prometheus"

	coreagent "github.com/juju/juju/agent"
	"github.com/juju/juju/agent/engine"
	"github.com/juju/juju/api"
	"github.com/juju/juju/api/base"
	"github.com/juju/juju/api/controller/crosscontroller"
	"github.com/juju/juju/caas"
	"github.com/juju/juju/core/instance"
	"github.com/juju/juju/core/machinelock"
	"github.com/juju/juju/core/presence"
	containerbroker "github.com/juju/juju/internal/container/broker"
	"github.com/juju/juju/internal/container/lxd"
	proxyconfig "github.com/juju/juju/internal/proxy/config"
	"github.com/juju/juju/internal/servicefactory"
	"github.com/juju/juju/state"
	"github.com/juju/juju/upgrades"
	jworker "github.com/juju/juju/worker"
	"github.com/juju/juju/worker/agent"
	"github.com/juju/juju/worker/agentconfigupdater"
	"github.com/juju/juju/worker/apiaddressupdater"
	"github.com/juju/juju/worker/apicaller"
	"github.com/juju/juju/worker/apiconfigwatcher"
	"github.com/juju/juju/worker/apiserver"
	"github.com/juju/juju/worker/apiservercertwatcher"
	"github.com/juju/juju/worker/auditconfigupdater"
	"github.com/juju/juju/worker/authenticationworker"
	"github.com/juju/juju/worker/caasunitsmanager"
	"github.com/juju/juju/worker/caasupgrader"
	"github.com/juju/juju/worker/centralhub"
	"github.com/juju/juju/worker/certupdater"
	"github.com/juju/juju/worker/changestream"
	"github.com/juju/juju/worker/changestreampruner"
	"github.com/juju/juju/worker/common"
	lxdbroker "github.com/juju/juju/worker/containerbroker"
	"github.com/juju/juju/worker/controlleragentconfig"
	"github.com/juju/juju/worker/credentialvalidator"
	"github.com/juju/juju/worker/dbaccessor"
	"github.com/juju/juju/worker/deployer"
	"github.com/juju/juju/worker/diskmanager"
	"github.com/juju/juju/worker/externalcontrollerupdater"
	"github.com/juju/juju/worker/fanconfigurer"
	"github.com/juju/juju/worker/filenotifywatcher"
	"github.com/juju/juju/worker/fortress"
	"github.com/juju/juju/worker/gate"
	"github.com/juju/juju/worker/hostkeyreporter"
	"github.com/juju/juju/worker/httpserver"
	"github.com/juju/juju/worker/httpserverargs"
	"github.com/juju/juju/worker/identityfilewriter"
	"github.com/juju/juju/worker/instancemutater"
	leasemanager "github.com/juju/juju/worker/lease"
	"github.com/juju/juju/worker/leaseexpiry"
	"github.com/juju/juju/worker/logger"
	"github.com/juju/juju/worker/logsender"
	"github.com/juju/juju/worker/machineactions"
	"github.com/juju/juju/worker/machiner"
	"github.com/juju/juju/worker/migrationflag"
	"github.com/juju/juju/worker/migrationminion"
	"github.com/juju/juju/worker/modelworkermanager"
	"github.com/juju/juju/worker/multiwatcher"
	"github.com/juju/juju/worker/peergrouper"
	prworker "github.com/juju/juju/worker/presence"
	"github.com/juju/juju/worker/provisioner"
	"github.com/juju/juju/worker/proxyupdater"
	psworker "github.com/juju/juju/worker/pubsub"
	"github.com/juju/juju/worker/querylogger"
	"github.com/juju/juju/worker/reboot"
	"github.com/juju/juju/worker/secretbackendrotate"
	workersf "github.com/juju/juju/worker/servicefactory"
	"github.com/juju/juju/worker/singular"
	workerstate "github.com/juju/juju/worker/state"
	"github.com/juju/juju/worker/stateconfigwatcher"
	"github.com/juju/juju/worker/stateconverter"
	"github.com/juju/juju/worker/storageprovisioner"
	"github.com/juju/juju/worker/syslogger"
	"github.com/juju/juju/worker/terminationworker"
	"github.com/juju/juju/worker/toolsversionchecker"
	"github.com/juju/juju/worker/trace"
	"github.com/juju/juju/worker/upgradedatabase"
	"github.com/juju/juju/worker/upgrader"
	"github.com/juju/juju/worker/upgradeseries"
	"github.com/juju/juju/worker/upgradesteps"
)

// ManifoldsConfig allows specialisation of the result of Manifolds.
type ManifoldsConfig struct {

	// AgentName is the name of the machine agent, like "machine-12".
	// This will never change during the execution of an agent, and
	// is used to provide this as config into a worker rather than
	// making the worker get it from the agent worker itself.
	AgentName string

	// Agent contains the agent that will be wrapped and made available to
	// its dependencies via a dependency.Engine.
	Agent coreagent.Agent

	// AgentConfigChanged is set whenever the machine agent's config
	// is updated.
	AgentConfigChanged *voyeur.Value

	// RootDir is the root directory that any worker that needs to
	// access local filesystems should use as a base. In actual use it
	// will be "" but it may be overridden in tests.
	RootDir string

	// PreviousAgentVersion passes through the version the machine
	// agent was running before the current restart.
	PreviousAgentVersion version.Number

	// UpgradeDBLock is passed to the upgrade database gate to
	// coordinate workers that shouldn't do anything until the
	// upgrade-database worker is done.
	UpgradeDBLock gate.Lock

	// UpgradeStepsLock is passed to the upgrade steps gate to
	// coordinate workers that shouldn't do anything until the
	// upgrade-steps worker is done.
	UpgradeStepsLock gate.Lock

	// UpgradeCheckLock is passed to the upgrade check gate to
	// coordinate workers that shouldn't do anything until the
	// upgrader worker completes it's first check.
	UpgradeCheckLock gate.Lock

	// NewDBWorkerFunc returns a tracked db worker.
	NewDBWorkerFunc dbaccessor.NewDBWorkerFunc

	// OpenStatePool is function used by the state manifold to create a
	// *state.StatePool.
	OpenStatePool func(stdcontext.Context, coreagent.Config, servicefactory.ControllerServiceFactory) (*state.StatePool, error)

	// OpenStateForUpgrade is a function the upgradesteps worker can
	// use to establish a connection to state.
	OpenStateForUpgrade func() (*state.StatePool, upgradesteps.SystemState, error)

	// MachineStartup is passed to the machine manifold. It does
	// machine setup work which relies on an API connection.
	MachineStartup func(api.Connection, Logger) error

	// PreUpgradeSteps is a function that is used by the upgradesteps
	// worker to ensure that conditions are OK for an upgrade to
	// proceed.
	PreUpgradeSteps upgrades.PreUpgradeStepsFunc

	// LogSource defines the channel type used to send log message
	// structs within the machine agent.
	LogSource logsender.LogRecordCh

	// NewDeployContext gives the tests the opportunity to create a
	// deployer.Context that can be used for testing.
	NewDeployContext func(deployer.ContextConfig) (deployer.Context, error)

	// Clock supplies timekeeping services to various workers.
	Clock clock.Clock

	// ValidateMigration is called by the migrationminion during the
	// migration process to check that the agent will be ok when
	// connected to the new target controller.
	ValidateMigration func(base.APICaller) error

	// PrometheusRegisterer is a prometheus.Registerer that may be used
	// by workers to register Prometheus metric collectors.
	PrometheusRegisterer prometheus.Registerer

	// CentralHub is the primary hub that exists in the apiserver.
	CentralHub *pubsub.StructuredHub

	// LocalHub is a simple pubsub that is used for internal agent
	// messaging only. This is used for interactions between workers
	// and the introspection worker.
	LocalHub *pubsub.SimpleHub

	// PubSubReporter is the introspection reporter for the pubsub forwarding
	// worker.
	PubSubReporter psworker.Reporter

	// PresenceRecorder
	PresenceRecorder presence.Recorder

	// UpdateLoggerConfig is a function that will save the specified
	// config value as the logging config in the agent.conf file.
	UpdateLoggerConfig func(string) error

	// NewAgentStatusSetter provides upgradesteps.StatusSetter.
	NewAgentStatusSetter func(apiConn api.Connection) (upgradesteps.StatusSetter, error)

	// ControllerLeaseDuration defines for how long this agent will ask
	// for controller administration rights.
	ControllerLeaseDuration time.Duration

	// TransactionPruneInterval defines how frequently mgo/txn transactions
	// are pruned from the database.
	TransactionPruneInterval time.Duration

	// SetStatePool is used by the state worker for informing the agent of
	// the StatePool that it creates, so we can pass it to the introspection
	// worker running outside of the dependency engine.
	SetStatePool func(*state.StatePool)

	// RegisterIntrospectionHTTPHandlers is a function that calls the
	// supplied function to register introspection HTTP handlers. The
	// function will be passed a path and a handler; the function may
	// alter the path as it sees fit, e.g. by adding a prefix.
	RegisterIntrospectionHTTPHandlers func(func(path string, _ http.Handler))

	// NewModelWorker returns a new worker for managing the model with
	// the specified UUID and type.
	NewModelWorker modelworkermanager.NewModelWorkerFunc

	// MachineLock is a central source for acquiring the machine lock.
	// This is used by a number of workers to ensure serialisation of actions
	// across the machine.
	MachineLock machinelock.Lock

	// MuxShutdownWait is the maximum time the http-server worker will wait
	// for all mux clients to gracefully terminate before the http-worker
	// exits regardless.
	MuxShutdownWait time.Duration

	// NewContainerBrokerFunc is a function opens a CAAS provider.
	NewContainerBrokerFunc caas.NewContainerBrokerFunc

	// NewBrokerFunc is a function opens a instance broker (LXD/KVM)
	NewBrokerFunc containerbroker.NewBrokerFunc

	// IsCaasConfig is true if this config is for a caas agent.
	IsCaasConfig bool

	// UnitEngineConfig is used by the deployer to initialize the unit's
	// dependency engine when running in the nested context.
	UnitEngineConfig func() dependency.EngineConfig

	// SetupLogging is used by the deployer to initialize the logging
	// context for the unit.
	SetupLogging func(*loggo.Context, coreagent.Config)

	// DependencyEngineMetrics creates a set of metrics for a model, so it's
	// possible to know the lifecycle of the workers in the dependency engine.
	DependencyEngineMetrics modelworkermanager.ModelMetrics

	// CharmhubHTTPClient is the HTTP client used for Charmhub API requests.
	CharmhubHTTPClient HTTPClient
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// commonManifolds returns a set of co-configured manifolds covering the
// various responsibilities of a machine agent.
//
// Thou Shalt Not Use String Literals In This Function. Or Else.
func commonManifolds(config ManifoldsConfig) dependency.Manifolds {

	// connectFilter exists:
	//  1) to let us retry api connections immediately on password change,
	//     rather than causing the dependency engine to wait for a while;
	//  2) to decide how to deal with fatal, non-recoverable errors
	//     e.g. apicaller.ErrConnectImpossible.
	connectFilter := func(err error) error {
		cause := errors.Cause(err)
		if cause == apicaller.ErrConnectImpossible {
			return jworker.ErrTerminateAgent
		} else if cause == apicaller.ErrChangedPassword {
			return dependency.ErrBounce
		}
		return err
	}

	newExternalControllerWatcherClient := func(apiInfo *api.Info) (
		externalcontrollerupdater.ExternalControllerWatcherClientCloser, error,
	) {
		conn, err := apicaller.NewExternalControllerConnection(apiInfo)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return crosscontroller.NewClient(conn), nil
	}

	var externalUpdateProxyFunc func(proxy.Settings) error
	if runtime.GOOS == "linux" && !config.IsCaasConfig {
		externalUpdateProxyFunc = lxd.ConfigureLXDProxies
	}

	agentConfig := config.Agent.CurrentConfig()
	agentTag := agentConfig.Tag()
	controllerTag := agentConfig.Controller()

	manifolds := dependency.Manifolds{
		// The agent manifold references the enclosing agent, and is the
		// foundation stone on which most other manifolds ultimately depend.
		agentName: agent.Manifold(config.Agent),

		// The termination worker returns ErrTerminateAgent if a
		// termination signal is received by the process it's running
		// in. It has no inputs and its only output is the error it
		// returns. It depends on the uninstall file having been
		// written *by the manual provider* at install time; it would
		// be Very Wrong Indeed to use SetCanUninstall in conjunction
		// with this code.
		terminationName: terminationworker.Manifold(),

		clockName: clockManifold(config.Clock),

		// Each machine agent has a flag manifold/worker which
		// reports whether or not the agent is a controller.
		isControllerFlagName: isControllerFlagManifold(true),

		// Controller agent config manifold watches the controller
		// agent config and bounces if it changes.
		controllerAgentConfigName: ifController(controlleragentconfig.Manifold(controlleragentconfig.ManifoldConfig{
			Clock:  config.Clock,
			Logger: loggo.GetLogger("juju.worker.controlleragentconfig"),
		})),

		// The stateconfigwatcher manifold watches the machine agent's
		// configuration and reports if state serving info is
		// present. It will bounce itself if state serving info is
		// added or removed. It is intended as a dependency just for
		// the state manifold.
		stateConfigWatcherName: stateconfigwatcher.Manifold(stateconfigwatcher.ManifoldConfig{
			AgentName:          agentName,
			AgentConfigChanged: config.AgentConfigChanged,
		}),

		// The centralhub manifold watches the state config to make sure it
		// only starts for machines that are api servers. Currently the hub is
		// passed in as config, but when the apiserver and peergrouper are
		// updated to use the dependency engine, the centralhub manifold
		// should also take the agentName so the worker can get the machine ID
		// for the creation of the hub.
		centralHubName: centralhub.Manifold(centralhub.ManifoldConfig{
			StateConfigWatcherName: stateConfigWatcherName,
			Hub:                    config.CentralHub,
		}),

		// The pubsub manifold gets the APIInfo from the agent config,
		// and uses this as a basis to talk to the other API servers.
		// The worker subscribes to the messages sent by the peergrouper
		// that defines the set of machines that are the API servers.
		// All non-local messages that originate from the machine that
		// is running the worker get forwarded to the other API servers.
		// This worker does not run in non-API server machines through
		// the hub dependency, as that is only available if the machine
		// is an API server.
		pubSubName: psworker.Manifold(psworker.ManifoldConfig{
			AgentName:      agentName,
			CentralHubName: centralHubName,
			Clock:          config.Clock,
			Logger:         loggo.GetLogger("juju.worker.pubsub"),
			NewWorker:      psworker.NewWorker,
			Reporter:       config.PubSubReporter,
		}),

		// The presence manifold listens to pubsub messages about the pubsub
		// forwarding connections and api connection and disconnections to
		// establish a view on which agents are "alive".
		presenceName: prworker.Manifold(prworker.ManifoldConfig{
			AgentName: agentName,
			// CentralHubName depends on StateConfigWatcherName,
			// which implies this can only run on controllers.
			CentralHubName: centralHubName,
			Recorder:       config.PresenceRecorder,
			Logger:         loggo.GetLogger("juju.worker.presence"),
			NewWorker:      prworker.NewWorker,
		}),

		// The state manifold creates a *state.State and makes it
		// available to other manifolds. It pings the mongodb session
		// regularly and will die if pings fail.
		stateName: workerstate.Manifold(workerstate.ManifoldConfig{
			AgentName:              agentName,
			StateConfigWatcherName: stateConfigWatcherName,
			ServiceFactoryName:     serviceFactoryName,
			OpenStatePool:          config.OpenStatePool,
			SetStatePool:           config.SetStatePool,
		}),

		// The multiwatcher manifold watches all the changes in the database
		// through the AllWatcherBacking and manages notifying the multiwatchers.
		multiwatcherName: ifController(multiwatcher.Manifold(multiwatcher.ManifoldConfig{
			StateName:            stateName,
			Clock:                config.Clock,
			Logger:               loggo.GetLogger("juju.worker.multiwatcher"),
			PrometheusRegisterer: config.PrometheusRegisterer,
			NewWorker:            multiwatcher.NewWorkerShim,
			NewAllWatcher:        state.NewAllWatcherBacking,
		})),

		// The api-config-watcher manifold monitors the API server
		// addresses in the agent config and bounces when they
		// change. It's required as part of model migrations.
		apiConfigWatcherName: apiconfigwatcher.Manifold(apiconfigwatcher.ManifoldConfig{
			AgentName:          agentName,
			AgentConfigChanged: config.AgentConfigChanged,
			Logger:             loggo.GetLogger("juju.worker.apiconfigwatcher"),
		}),

		// The certificate-watcher manifold monitors the API server
		// certificate in the agent config for changes, and parses
		// and offers the result to other manifolds. This is only
		// run by state servers.
		certificateWatcherName: ifController(apiservercertwatcher.Manifold(apiservercertwatcher.ManifoldConfig{
			AgentName: agentName,
		})),

		// The api caller is a thin concurrent wrapper around a connection
		// to some API server. It's used by many other manifolds, which all
		// select their own desired facades. It will be interesting to see
		// how this works when we consolidate the agents; might be best to
		// handle the auth changes server-side..?
		apiCallerName: apicaller.Manifold(apicaller.ManifoldConfig{
			AgentName:            agentName,
			APIConfigWatcherName: apiConfigWatcherName,
			APIOpen:              api.Open,
			NewConnection:        apicaller.ScaryConnect,
			Filter:               connectFilter,
			Logger:               loggo.GetLogger("juju.worker.apicaller"),
		}),

		// The upgrade database gate is used to coordinate workers that should
		// not do anything until the upgrade-database worker has finished
		// running any required database upgrade steps.
		upgradeDatabaseGateName: ifController(gate.ManifoldEx(config.UpgradeDBLock)),
		upgradeDatabaseFlagName: ifController(gate.FlagManifold(gate.FlagManifoldConfig{
			GateName:  upgradeDatabaseGateName,
			NewWorker: gate.NewFlagWorker,
		})),

		// The upgrade-database worker runs soon after the machine agent starts
		// and runs any steps required to upgrade to the database to the
		// current version. Once upgrade steps have run, the upgrade-database
		// gate is unlocked and the worker exits.
		upgradeDatabaseName: ifController(upgradedatabase.Manifold(upgradedatabase.ManifoldConfig{
			AgentName:          agentName,
			UpgradeDBGateName:  upgradeDatabaseGateName,
			DBAccessorName:     dbAccessorName,
			ServiceFactoryName: serviceFactoryName,
			NewWorker:          upgradedatabase.NewUpgradeDatabaseWorker,
			Logger:             loggo.GetLogger("juju.worker.upgradedatabase"),
			Clock:              config.Clock,
		})),

		// The upgrade steps gate is used to coordinate workers which
		// shouldn't do anything until the upgrade-steps worker has
		// finished running any required upgrade steps. The flag of
		// similar name is used to implement the isFullyUpgraded func
		// that keeps upgrade concerns out of unrelated manifolds.
		upgradeStepsGateName: gate.ManifoldEx(config.UpgradeStepsLock),
		upgradeStepsFlagName: gate.FlagManifold(gate.FlagManifoldConfig{
			GateName:  upgradeStepsGateName,
			NewWorker: gate.NewFlagWorker,
		}),

		// The upgrade check gate is used to coordinate workers which
		// shouldn't do anything until the upgrader worker has
		// completed its first check for a new tools version to
		// upgrade to. The flag of similar name is used to implement
		// the isFullyUpgraded func that keeps upgrade concerns out of
		// unrelated manifolds.
		upgradeCheckGateName: gate.ManifoldEx(config.UpgradeCheckLock),
		upgradeCheckFlagName: gate.FlagManifold(gate.FlagManifoldConfig{
			GateName:  upgradeCheckGateName,
			NewWorker: gate.NewFlagWorker,
		}),

		// The upgradesteps worker runs soon after the machine agent
		// starts and runs any steps required to upgrade to the
		// running jujud version. Once upgrade steps have run, the
		// upgradesteps gate is unlocked and the worker exits.
		upgradeStepsName: upgradesteps.Manifold(upgradesteps.ManifoldConfig{
			AgentName:            agentName,
			APICallerName:        apiCallerName,
			UpgradeStepsGateName: upgradeStepsGateName,
			OpenStateForUpgrade:  config.OpenStateForUpgrade,
			PreUpgradeSteps:      config.PreUpgradeSteps,
			NewAgentStatusSetter: config.NewAgentStatusSetter,
		}),

		// The migration workers collaborate to run migrations;
		// and to create a mechanism for running other workers
		// so they can't accidentally interfere with a migration
		// in progress. Such a manifold should (1) depend on the
		// migration-inactive flag, to know when to start or die;
		// and (2) occupy the migration-fortress, so as to avoid
		// possible interference with the minion (which will not
		// take action until it's gained sole control of the
		// fortress).
		//
		// Note that the fortress itself will not be created
		// until the upgrade process is complete; this frees all
		// its dependencies from upgrade concerns.
		migrationFortressName: ifFullyUpgraded(fortress.Manifold()),
		migrationInactiveFlagName: migrationflag.Manifold(migrationflag.ManifoldConfig{
			APICallerName: apiCallerName,
			Check:         migrationflag.IsTerminal,
			NewFacade:     migrationflag.NewFacade,
			NewWorker:     migrationflag.NewWorker,
		}),
		migrationMinionName: migrationminion.Manifold(migrationminion.ManifoldConfig{
			AgentName:         agentName,
			APICallerName:     apiCallerName,
			FortressName:      migrationFortressName,
			Clock:             config.Clock,
			APIOpen:           api.Open,
			ValidateMigration: config.ValidateMigration,
			NewFacade:         migrationminion.NewFacade,
			NewWorker:         migrationminion.NewWorker,
			Logger:            loggo.GetLogger("juju.worker.migrationminion"),
		}),

		// Each controller machine runs a singular worker which will
		// attempt to claim responsibility for running certain workers
		// that must not be run concurrently by multiple agents.
		isPrimaryControllerFlagName: ifController(singular.Manifold(singular.ManifoldConfig{
			Clock:         config.Clock,
			APICallerName: apiCallerName,
			Duration:      config.ControllerLeaseDuration,
			Claimant:      agentTag,
			Entity:        controllerTag,
			NewFacade:     singular.NewFacade,
			NewWorker:     singular.NewWorker,
		})),

		// The agent-config-updater manifold sets the state serving info from
		// the API connection and writes it to the agent config.
		agentConfigUpdaterName: ifNotMigrating(agentconfigupdater.Manifold(agentconfigupdater.ManifoldConfig{
			AgentName:      agentName,
			APICallerName:  apiCallerName,
			CentralHubName: centralHubName,
			Logger:         loggo.GetLogger("juju.worker.agentconfigupdater"),
		})),

		// The logging config updater is a leaf worker that indirectly
		// controls the messages sent via the log sender or rsyslog,
		// according to changes in environment config. We should only need
		// one of these in a consolidated agent.
		loggingConfigUpdaterName: ifNotMigrating(logger.Manifold(logger.ManifoldConfig{
			AgentName:       agentName,
			APICallerName:   apiCallerName,
			LoggingContext:  loggo.DefaultContext(),
			Logger:          loggo.GetLogger("juju.worker.logger"),
			UpdateAgentFunc: config.UpdateLoggerConfig,
		})),

		// The log sender is a leaf worker that sends log messages to some
		// API server, when configured so to do. We should only need one of
		// these in a consolidated agent.
		//
		// NOTE: the LogSource will buffer a large number of messages as an upgrade
		// runs; it currently seems better to fill the buffer and send when stable,
		// optimising for stable controller upgrades rather than up-to-the-moment
		// observable normal-machine upgrades.
		logSenderName: ifNotMigrating(logsender.Manifold(logsender.ManifoldConfig{
			APICallerName: apiCallerName,
			LogSource:     config.LogSource,
		})),

		identityFileWriterName: ifNotMigrating(identityfilewriter.Manifold(identityfilewriter.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
		})),

		externalControllerUpdaterName: ifNotMigrating(ifPrimaryController(externalcontrollerupdater.Manifold(
			externalcontrollerupdater.ManifoldConfig{
				APICallerName:                      apiCallerName,
				NewExternalControllerWatcherClient: newExternalControllerWatcherClient,
			},
		))),

		traceName: trace.Manifold(trace.ManifoldConfig{
			AgentName:       agentName,
			Clock:           config.Clock,
			Logger:          loggo.GetLogger("juju.worker.trace"),
			NewTracerWorker: trace.NewTracerWorker,
		}),

		httpServerArgsName: httpserverargs.Manifold(httpserverargs.ManifoldConfig{
			ClockName:             clockName,
			StateName:             stateName,
			ServiceFactoryName:    serviceFactoryName,
			NewStateAuthenticator: httpserverargs.NewStateAuthenticator,
		}),

		httpServerName: httpserver.Manifold(httpserver.ManifoldConfig{
			AuthorityName:        certificateWatcherName,
			HubName:              centralHubName,
			StateName:            stateName,
			ServiceFactoryName:   serviceFactoryName,
			MuxName:              httpServerArgsName,
			APIServerName:        apiServerName,
			PrometheusRegisterer: config.PrometheusRegisterer,
			AgentName:            config.AgentName,
			Clock:                config.Clock,
			MuxShutdownWait:      config.MuxShutdownWait,
			LogDir:               agentConfig.LogDir(),
			Logger:               loggo.GetLogger("juju.worker.httpserver"),
			GetControllerConfig:  httpserver.GetControllerConfig,
			NewTLSConfig:         httpserver.NewTLSConfig,
			NewWorker:            httpserver.NewWorkerShim,
		}),

		apiServerName: apiserver.Manifold(apiserver.ManifoldConfig{
			AgentName:              agentName,
			AuthenticatorName:      httpServerArgsName,
			ClockName:              clockName,
			StateName:              stateName,
			SyslogName:             syslogName,
			MultiwatcherName:       multiwatcherName,
			MuxName:                httpServerArgsName,
			LeaseManagerName:       leaseManagerName,
			UpgradeGateName:        upgradeStepsGateName,
			AuditConfigUpdaterName: auditConfigUpdaterName,
			CharmhubHTTPClientName: charmhubHTTPClientName,
			TraceName:              traceName,

			// Note that although there is a transient dependency on dbaccessor
			// via changestream, the direct dependency supplies the capability
			// to remove databases corresponding to destroyed/migrated models.
			ServiceFactoryName: serviceFactoryName,
			ChangeStreamName:   changeStreamName,

			PrometheusRegisterer:              config.PrometheusRegisterer,
			RegisterIntrospectionHTTPHandlers: config.RegisterIntrospectionHTTPHandlers,
			Hub:                               config.CentralHub,
			Presence:                          config.PresenceRecorder,
			NewWorker:                         apiserver.NewWorker,
			NewMetricsCollector:               apiserver.NewMetricsCollector,
		}),

		charmhubHTTPClientName: dependency.Manifold{
			Start: func(_ dependency.Context) (worker.Worker, error) {
				return engine.NewValueWorker(config.CharmhubHTTPClient)
			},
			Output: engine.ValueWorkerOutput,
		},

		modelWorkerManagerName: ifFullyUpgraded(modelworkermanager.Manifold(modelworkermanager.ManifoldConfig{
			AgentName:          agentName,
			AuthorityName:      certificateWatcherName,
			StateName:          stateName,
			SyslogName:         syslogName,
			ServiceFactoryName: serviceFactoryName,
			Clock:              config.Clock,
			MuxName:            httpServerArgsName,
			NewWorker:          modelworkermanager.New,
			NewModelWorker:     config.NewModelWorker,
			ModelMetrics:       config.DependencyEngineMetrics,
			Logger:             loggo.GetLogger("juju.workers.modelworkermanager"),
		})),

		peergrouperName: ifFullyUpgraded(peergrouper.Manifold(peergrouper.ManifoldConfig{
			AgentName:            agentName,
			ClockName:            clockName,
			StateName:            stateName,
			Hub:                  config.CentralHub,
			PrometheusRegisterer: config.PrometheusRegisterer,
			NewWorker:            peergrouper.New,
		})),

		serviceFactoryName: workersf.Manifold(workersf.ManifoldConfig{
			DBAccessorName:              dbAccessorName,
			ChangeStreamName:            changeStreamName,
			Logger:                      workersf.NewLogger("juju.worker.servicefactory"),
			NewWorker:                   workersf.NewWorker,
			NewServiceFactoryGetter:     workersf.NewServiceFactoryGetter,
			NewControllerServiceFactory: workersf.NewControllerServiceFactory,
			NewModelServiceFactory:      workersf.NewModelServiceFactory,
		}),

		dbAccessorName: ifController(dbaccessor.Manifold(dbaccessor.ManifoldConfig{
			AgentName:            agentName,
			QueryLoggerName:      queryLoggerName,
			Clock:                config.Clock,
			Hub:                  config.CentralHub,
			Logger:               loggo.GetLogger("juju.worker.dbaccessor"),
			LogDir:               agentConfig.LogDir(),
			PrometheusRegisterer: config.PrometheusRegisterer,
			NewApp:               dbaccessor.NewApp,
			NewDBWorker:          config.NewDBWorkerFunc,
			NewMetricsCollector:  dbaccessor.NewMetricsCollector,
		})),

		queryLoggerName: ifController(querylogger.Manifold(querylogger.ManifoldConfig{
			LogDir: agentConfig.LogDir(),
			Clock:  config.Clock,
			Logger: loggo.GetLogger("juju.worker.querylogger"),
		})),

		fileNotifyWatcherName: ifController(filenotifywatcher.Manifold(filenotifywatcher.ManifoldConfig{
			Clock:             config.Clock,
			Logger:            loggo.GetLogger("juju.worker.filenotifywatcher"),
			NewWatcher:        filenotifywatcher.NewWatcher,
			NewINotifyWatcher: filenotifywatcher.NewINotifyWatcher,
		})),

		changeStreamName: ifController(changestream.Manifold(changestream.ManifoldConfig{
			AgentName:            agentName,
			DBAccessor:           dbAccessorName,
			FileNotifyWatcher:    fileNotifyWatcherName,
			Clock:                config.Clock,
			Logger:               loggo.GetLogger("juju.worker.changestream"),
			PrometheusRegisterer: config.PrometheusRegisterer,
			NewWatchableDB:       changestream.NewWatchableDB,
			NewMetricsCollector:  changestream.NewMetricsCollector,
		})),

		changeStreamPrunerName: ifPrimaryController(changestreampruner.Manifold(changestreampruner.ManifoldConfig{
			DBAccessor: dbAccessorName,
			Clock:      config.Clock,
			Logger:     loggo.GetLogger("juju.worker.changestreampruner"),
			NewWorker:  changestreampruner.NewWorker,
		})),

		auditConfigUpdaterName: ifController(auditconfigupdater.Manifold(auditconfigupdater.ManifoldConfig{
			AgentName: agentName,
			StateName: stateName,
			NewWorker: auditconfigupdater.New,
		})),

		// The lease expiry worker constantly deletes
		// leases with an expiry time in the past.
		leaseExpiryName: ifPrimaryController(leaseexpiry.Manifold(leaseexpiry.ManifoldConfig{
			ClockName:      clockName,
			DBAccessorName: dbAccessorName,
			Logger:         loggo.GetLogger("juju.worker.leaseexpiry"),
			NewWorker:      leaseexpiry.NewWorker,
			NewStore:       leaseexpiry.NewStore,
		})),

		// The global lease manager tracks lease information in the Dqlite database.
		leaseManagerName: ifController(leasemanager.Manifold(leasemanager.ManifoldConfig{
			AgentName:            agentName,
			ClockName:            clockName,
			DBAccessorName:       dbAccessorName,
			Logger:               loggo.GetLogger("juju.worker.lease"),
			LogDir:               agentConfig.LogDir(),
			PrometheusRegisterer: config.PrometheusRegisterer,
			NewWorker:            leasemanager.NewWorker,
			NewStore:             leasemanager.NewStore,
		})),

		// The proxy config updater is a leaf worker that sets http/https/apt/etc
		// proxy settings.
		proxyConfigUpdater: ifNotMigrating(proxyupdater.Manifold(proxyupdater.ManifoldConfig{
			AgentName:           agentName,
			APICallerName:       apiCallerName,
			Logger:              loggo.GetLogger("juju.worker.proxyupdater"),
			WorkerFunc:          proxyupdater.NewWorker,
			SupportLegacyValues: !config.IsCaasConfig,
			ExternalUpdate:      externalUpdateProxyFunc,
			InProcessUpdate:     proxyconfig.DefaultConfig.Set,
			RunFunc:             proxyupdater.RunWithStdIn,
		})),

		// TODO (thumper): It doesn't really make sense in a machine manifold as
		// not every machine will have credentials. It is here for the
		// ifCredentialValid function that is used solely for the machine
		// storage provisioner. It isn't clear to me why we have a storage
		// provisioner in the machine agent and the model workers.
		validCredentialFlagName: credentialvalidator.Manifold(credentialvalidator.ManifoldConfig{
			APICallerName: apiCallerName,
			NewFacade:     credentialvalidator.NewFacade,
			NewWorker:     credentialvalidator.NewWorker,
			Logger:        loggo.GetLogger("juju.worker.credentialvalidator"),
		}),

		secretBackendRotateName: ifNotMigrating(ifPrimaryController(secretbackendrotate.Manifold(
			secretbackendrotate.ManifoldConfig{
				APICallerName: apiCallerName,
				Logger:        loggo.GetLogger("juju.worker.secretbackendsrotate"),
			},
		))),
	}

	return manifolds
}

// IAASManifolds returns a set of co-configured manifolds covering the
// various responsibilities of a IAAS machine agent.
func IAASManifolds(config ManifoldsConfig) dependency.Manifolds {
	manifolds := dependency.Manifolds{
		toolsVersionCheckerName: ifNotMigrating(toolsversionchecker.Manifold(toolsversionchecker.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
		})),

		authenticationWorkerName: ifNotMigrating(authenticationworker.Manifold(authenticationworker.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
		})),

		hostKeyReporterName: ifNotMigrating(hostkeyreporter.Manifold(hostkeyreporter.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
			RootDir:       config.RootDir,
			NewFacade:     hostkeyreporter.NewFacade,
			NewWorker:     hostkeyreporter.NewWorker,
		})),

		fanConfigurerName: ifNotMigrating(fanconfigurer.Manifold(fanconfigurer.ManifoldConfig{
			APICallerName: apiCallerName,
			Clock:         config.Clock,
		})),

		certificateUpdaterName: ifFullyUpgraded(certupdater.Manifold(certupdater.ManifoldConfig{
			AgentName:                agentName,
			AuthorityName:            certificateWatcherName,
			StateName:                stateName,
			ServiceFactoryName:       serviceFactoryName,
			NewWorker:                certupdater.NewCertificateUpdater,
			NewMachineAddressWatcher: certupdater.NewMachineAddressWatcher,
			Logger:                   loggo.GetLogger("juju.worker.certupdater"),
		})),

		// The machiner Worker will wait for the identified machine to become
		// Dying and make it Dead; or until the machine becomes Dead by other
		// means. This worker needs to be launched after fanconfigurer
		// so that it reports interfaces created by it.
		machinerName: ifNotMigrating(machiner.Manifold(machiner.ManifoldConfig{
			AgentName:         agentName,
			APICallerName:     apiCallerName,
			FanConfigurerName: fanConfigurerName,
		})),

		// The diskmanager worker periodically lists block devices on the
		// machine it runs on. This worker will be run on all Juju-managed
		// machines (one per machine agent).
		diskManagerName: ifNotMigrating(diskmanager.Manifold(diskmanager.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
		})),

		// The api address updater is a leaf worker that rewrites agent config
		// as the state server addresses change. We should only need one of
		// these in a consolidated agent.
		apiAddressUpdaterName: ifNotMigrating(apiaddressupdater.Manifold(apiaddressupdater.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
			Logger:        loggo.GetLogger("juju.worker.apiaddressupdater"),
		})),

		machineActionName: ifNotMigrating(machineactions.Manifold(machineactions.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
			NewFacade:     machineactions.NewFacade,
			NewWorker:     machineactions.NewMachineActionsWorker,
			MachineLock:   config.MachineLock,
		})),

		// The upgrader is a leaf worker that returns a specific error
		// type recognised by the machine agent, causing other workers
		// to be stopped and the agent to be restarted running the new
		// tools. We should only need one of these in a consolidated
		// agent, but we'll need to be careful about behavioural
		// differences, and interactions with the upgrade-steps
		// worker.
		upgraderName: upgrader.Manifold(upgrader.ManifoldConfig{
			AgentName:            agentName,
			APICallerName:        apiCallerName,
			UpgradeStepsGateName: upgradeStepsGateName,
			UpgradeCheckGateName: upgradeCheckGateName,
			PreviousAgentVersion: config.PreviousAgentVersion,
			Logger:               loggo.GetLogger("juju.worker.upgrader"),
			Clock:                config.Clock,
		}),

		upgradeSeriesWorkerName: ifNotMigrating(upgradeseries.Manifold(upgradeseries.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
			Logger:        loggo.GetLogger("juju.worker.upgradeseries"),
			NewFacade:     upgradeseries.NewFacade,
			NewWorker:     upgradeseries.NewWorker,
		})),

		// The deployer worker is primarily for deploying and recalling unit
		// agents, according to changes in a set of state units; and for the
		// final removal of its agents' units from state when they are no
		// longer needed.
		deployerName: ifFullyUpgraded(deployer.Manifold(deployer.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
			Clock:         config.Clock,
			Hub:           config.LocalHub,
			Logger:        loggo.GetLogger("juju.worker.deployer"),

			UnitEngineConfig: config.UnitEngineConfig,
			SetupLogging:     config.SetupLogging,
			NewDeployContext: config.NewDeployContext,
		})),

		// The reboot manifold manages a worker which will reboot the
		// machine when requested. It needs an API connection and
		// waits for upgrades to be complete.
		rebootName: ifNotMigrating(reboot.Manifold(reboot.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
			MachineLock:   config.MachineLock,
		})),

		// The storageProvisioner worker manages provisioning
		// (deprovisioning), and attachment (detachment) of first-class
		// volumes and filesystems.
		storageProvisionerName: ifNotMigrating(ifCredentialValid(storageprovisioner.MachineManifold(storageprovisioner.MachineManifoldConfig{
			AgentName:                    agentName,
			APICallerName:                apiCallerName,
			Clock:                        config.Clock,
			Logger:                       loggo.GetLogger("juju.worker.storageprovisioner"),
			NewCredentialValidatorFacade: common.NewCredentialInvalidatorFacade,
		}))),
		brokerTrackerName: ifNotMigrating(lxdbroker.Manifold(lxdbroker.ManifoldConfig{
			APICallerName: apiCallerName,
			AgentName:     agentName,
			MachineLock:   config.MachineLock,
			NewBrokerFunc: config.NewBrokerFunc,
			NewTracker:    lxdbroker.NewWorkerTracker,
		})),
		instanceMutaterName: ifNotMigrating(instancemutater.MachineManifold(instancemutater.MachineManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
			BrokerName:    brokerTrackerName,
			Logger:        loggo.GetLogger("juju.worker.instancemutater.container"),
			NewClient:     instancemutater.NewClient,
			NewWorker:     instancemutater.NewContainerWorker,
		})),
		syslogName: syslogger.Manifold(syslogger.ManifoldConfig{
			NewWorker: syslogger.NewWorker,
			NewLogger: syslogger.NewSyslog,
		}),
		// The machineSetupName manifold runs small tasks required
		// to setup a machine, but requires the machine agent's API
		// connection. Once its work is complete, it stops.
		machineSetupName: ifNotMigrating(MachineStartupManifold(MachineStartupConfig{
			APICallerName:  apiCallerName,
			MachineStartup: config.MachineStartup,
			Logger:         loggo.GetLogger("juju.worker.machinesetup"),
		})),
		kvmContainerProvisioner: ifNotMigrating(provisioner.ContainerProvisioningManifold(provisioner.ContainerManifoldConfig{
			AgentName:                    agentName,
			APICallerName:                apiCallerName,
			Logger:                       loggo.GetLogger("juju.worker.kvmprovisioner"),
			MachineLock:                  config.MachineLock,
			NewCredentialValidatorFacade: common.NewCredentialInvalidatorFacade,
			ContainerType:                instance.KVM,
		})),
		lxdContainerProvisioner: ifNotMigrating(provisioner.ContainerProvisioningManifold(provisioner.ContainerManifoldConfig{
			AgentName:                    agentName,
			APICallerName:                apiCallerName,
			Logger:                       loggo.GetLogger("juju.worker.lxdprovisioner"),
			MachineLock:                  config.MachineLock,
			NewCredentialValidatorFacade: common.NewCredentialInvalidatorFacade,
			ContainerType:                instance.LXD,
		})),
		// isNotControllerFlagName is only used for the stateconverter,
		isNotControllerFlagName: isControllerFlagManifold(false),
		stateConverterName: ifNotController(ifNotMigrating(stateconverter.Manifold(stateconverter.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
			Logger:        loggo.GetLogger("juju.worker.stateconverter"),
		}))),
	}

	return mergeManifolds(config, manifolds)
}

// CAASManifolds returns a set of co-configured manifolds covering the
// various responsibilities of a CAAS machine agent.
func CAASManifolds(config ManifoldsConfig) dependency.Manifolds {
	return mergeManifolds(config, dependency.Manifolds{
		// TODO(caas) - when we support HA, only want this on primary
		upgraderName: caasupgrader.Manifold(caasupgrader.ManifoldConfig{
			AgentName:            agentName,
			APICallerName:        apiCallerName,
			UpgradeStepsGateName: upgradeStepsGateName,
			UpgradeCheckGateName: upgradeCheckGateName,
			PreviousAgentVersion: config.PreviousAgentVersion,
		}),

		syslogName: syslogger.Manifold(syslogger.ManifoldConfig{
			NewWorker: syslogger.NewWorker,
			NewLogger: syslogger.NewDiscard,
		}),

		// The CAAS units manager worker runs on CAAS agent and subscribes and handles unit topics on the localhub.
		caasUnitsManager: caasunitsmanager.Manifold(caasunitsmanager.ManifoldConfig{
			AgentName:     agentName,
			APICallerName: apiCallerName,
			Clock:         config.Clock,
			Logger:        loggo.GetLogger("juju.worker.caasunitsmanager"),
			Hub:           config.LocalHub,
		}),
	})
}

func mergeManifolds(config ManifoldsConfig, manifolds dependency.Manifolds) dependency.Manifolds {
	result := commonManifolds(config)
	for name, manifold := range manifolds {
		result[name] = manifold
	}
	return result
}

func clockManifold(clock clock.Clock) dependency.Manifold {
	return dependency.Manifold{
		Start: func(_ dependency.Context) (worker.Worker, error) {
			return engine.NewValueWorker(clock)
		},
		Output: engine.ValueWorkerOutput,
	}
}

var ifFullyUpgraded = engine.Housing{
	Flags: []string{
		upgradeStepsFlagName,
		upgradeCheckFlagName,
	},
}.Decorate

var ifNotMigrating = engine.Housing{
	Flags: []string{
		migrationInactiveFlagName,
	},
	Occupy: migrationFortressName,
}.Decorate

var ifPrimaryController = engine.Housing{
	Flags: []string{
		isPrimaryControllerFlagName,
	},
}.Decorate

var ifController = engine.Housing{
	Flags: []string{
		isControllerFlagName,
	},
}.Decorate

var ifNotController = engine.Housing{
	Flags: []string{
		isNotControllerFlagName,
	},
}.Decorate

var ifCredentialValid = engine.Housing{
	Flags: []string{
		validCredentialFlagName,
	},
}.Decorate

const (
	agentName              = "agent"
	agentConfigUpdaterName = "agent-config-updater"
	terminationName        = "termination-signal-handler"
	stateConfigWatcherName = "state-config-watcher"
	stateName              = "state"
	apiCallerName          = "api-caller"
	s3CallerName           = "s3-caller"
	apiConfigWatcherName   = "api-config-watcher"
	centralHubName         = "central-hub"
	presenceName           = "presence"
	pubSubName             = "pubsub-forwarder"
	clockName              = "clock"

	upgradeDatabaseName     = "upgrade-database-runner"
	upgradeDatabaseGateName = "upgrade-database-gate"
	upgradeDatabaseFlagName = "upgrade-database-flag"

	upgraderName         = "upgrader"
	upgradeStepsName     = "upgrade-steps-runner"
	upgradeStepsGateName = "upgrade-steps-gate"
	upgradeStepsFlagName = "upgrade-steps-flag"
	upgradeCheckGateName = "upgrade-check-gate"
	upgradeCheckFlagName = "upgrade-check-flag"

	migrationFortressName     = "migration-fortress"
	migrationInactiveFlagName = "migration-inactive-flag"
	migrationMinionName       = "migration-minion"

	machineSetupName              = "machine-setup"
	rebootName                    = "reboot-executor"
	loggingConfigUpdaterName      = "logging-config-updater"
	diskManagerName               = "disk-manager"
	proxyConfigUpdater            = "proxy-config-updater"
	apiAddressUpdaterName         = "api-address-updater"
	machinerName                  = "machiner"
	logSenderName                 = "log-sender"
	deployerName                  = "deployer"
	authenticationWorkerName      = "ssh-authkeys-updater"
	storageProvisionerName        = "storage-provisioner"
	identityFileWriterName        = "ssh-identity-writer"
	toolsVersionCheckerName       = "tools-version-checker"
	machineActionName             = "machine-action-runner"
	hostKeyReporterName           = "host-key-reporter"
	fanConfigurerName             = "fan-configurer"
	externalControllerUpdaterName = "external-controller-updater"
	isPrimaryControllerFlagName   = "is-primary-controller-flag"
	isControllerFlagName          = "is-controller-flag"
	isNotControllerFlagName       = "is-not-controller-flag"
	instanceMutaterName           = "instance-mutater"
	certificateWatcherName        = "certificate-watcher"
	modelWorkerManagerName        = "model-worker-manager"
	multiwatcherName              = "multiwatcher"
	peergrouperName               = "peer-grouper"
	dbAccessorName                = "db-accessor"
	queryLoggerName               = "query-logger"
	fileNotifyWatcherName         = "file-notify-watcher"
	changeStreamName              = "change-stream"
	changeStreamPrunerName        = "change-stream-pruner"
	certificateUpdaterName        = "certificate-updater"
	auditConfigUpdaterName        = "audit-config-updater"
	leaseExpiryName               = "lease-expiry"
	leaseManagerName              = "lease-manager"
	stateConverterName            = "state-converter"
	serviceFactoryName            = "service-factory"
	lxdContainerProvisioner       = "lxd-container-provisioner"
	kvmContainerProvisioner       = "kvm-container-provisioner"
	controllerAgentConfigName     = "controller-agent-config"

	secretBackendRotateName = "secret-backend-rotate"

	upgradeSeriesWorkerName = "upgrade-series"

	traceName = "trace"

	httpServerName     = "http-server"
	httpServerArgsName = "http-server-args"
	apiServerName      = "api-server"

	syslogName       = "syslog"
	caasUnitsManager = "caas-units-manager"

	validCredentialFlagName = "valid-credential-flag"

	brokerTrackerName = "broker-tracker"

	charmhubHTTPClientName = "charmhub-http-client"
)
