// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretmigrationminion

import (
	// "fmt"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/names/v4"
	"github.com/juju/worker/v3"
	"github.com/juju/worker/v3/catacomb"
	"github.com/kr/pretty"

	coresecrets "github.com/juju/juju/core/secrets"
	"github.com/juju/juju/core/watcher"
	// "github.com/juju/juju/worker/fortress"
	jujusecrets "github.com/juju/juju/secrets"
	"github.com/juju/juju/secrets/provider"
)

// logger is here to stop the desire of creating a package level logger.
// Don't do this, instead use the one passed as manifold config.
type logger interface{}

var _ logger = struct{}{}

// Logger represents the methods used by the worker to log information.
type Logger interface {
	Tracef(string, ...interface{})
	Debugf(string, ...interface{})
	Warningf(string, ...interface{})
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
	Criticalf(string, ...interface{})
}

type SecretManagerFacade interface {
	WatchSecretMigrationTasks(ownerTag names.Tag) (watcher.StringsWatcher, error)
	StartSecretMigrationTask(id string) error
	FailSecretMigrationTask(id string) error
	CompleteSecretMigrationTask(id string) error
	SecretMetadata() ([]coresecrets.SecretOwnerMetadata, error)
	GetRevisionContentInfo(uri *coresecrets.URI, revision int, pendingDelete bool) (*jujusecrets.ContentParams, error)
	GetSecretBackendConfig() (*provider.ModelBackendConfigInfo, error)
}

// Config defines the operation of the Worker.
type Config struct {
	SecretManagerFacade SecretManagerFacade
	// Guard     fortress.Guard
	Logger Logger
	Clock  clock.Clock

	IsLeader bool
	UnitTag  names.UnitTag

	SecretsBackendGetter func() (jujusecrets.BackendsClient, error)
}

// Validate returns an error if config cannot drive the Worker.
func (config Config) Validate() error {
	if config.SecretManagerFacade == nil {
		return errors.NotValidf("nil SecretManagerFacade")
	}
	// if config.Guard == nil {
	// 	return errors.NotValidf("nil Guard")
	// }
	if config.Logger == nil {
		return errors.NotValidf("nil Logger")
	}
	if config.Clock == nil {
		return errors.NotValidf("nil Clock")
	}
	return nil
}

// NewWorker returns a secretmigrationminion Worker backed by config, or an error.
func NewWorker(config Config) (worker.Worker, error) {
	config.Logger.Criticalf("NewWorker config %s", pretty.Sprint(config))
	if err := config.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	w := &Worker{
		config: config,
	}
	err := catacomb.Invoke(catacomb.Plan{
		Site: &w.catacomb,
		Work: w.loop,
	})
	return w, errors.Trace(err)
}

// Worker migrates secrets to the new backend when the model's secret backend has changed.
type Worker struct {
	catacomb catacomb.Catacomb
	config   Config

	_secretsBackends jujusecrets.BackendsClient
}

// TODO:
// Note: this worker should be run for each unit Tag for unit owned secrets and application Tag for application owned secrets.

// Kill is defined on worker.Worker.
func (w *Worker) Kill() {
	w.catacomb.Kill(nil)
}

// Wait is part of the worker.Worker interface.
func (w *Worker) Wait() error {
	return w.catacomb.Wait()
}

func (w *Worker) secretsBackends() (_ jujusecrets.BackendsClient, err error) {
	if w._secretsBackends == nil {
		if w._secretsBackends, err = w.config.SecretsBackendGetter(); err != nil {
			return nil, errors.Trace(err)
		}
	}
	return w._secretsBackends, nil
}

func (w *Worker) loop() (err error) {
	watcherForUnit, err := w.config.SecretManagerFacade.WatchSecretMigrationTasks(w.config.UnitTag)
	if err != nil {
		return errors.Trace(err)
	}
	if err := w.catacomb.Add(watcherForUnit); err != nil {
		return errors.Trace(err)
	}
	var watcherForApplicationChan watcher.StringsChannel
	if w.config.IsLeader {
		watcherForApplication, err := w.config.SecretManagerFacade.WatchSecretMigrationTasks(appTag(w.config.UnitTag))
		if err != nil {
			return errors.Trace(err)
		}
		if err := w.catacomb.Add(watcherForApplication); err != nil {
			return errors.Trace(err)
		}
		watcherForApplicationChan = watcherForApplication.Changes()
	}

	for {
		select {
		case <-w.catacomb.Dying():
			return w.catacomb.ErrDying()
		case ids, ok := <-watcherForUnit.Changes():
			if !ok {
				return errors.New("secret migration tasks watcher for unit closed")
			}
			w.config.Logger.Criticalf("secret migration tasks for unit %v received!!!", ids)
			if err := w.migrateSecrets(
				func(ownerTag names.Tag) bool {
					w.config.Logger.Criticalf("migrateSecrets ownerTag %q, names.UnitTagKind?", ownerTag)
					return ownerTag.Kind() == names.UnitTagKind
				},
			); err != nil {
				return errors.Trace(err)
			}
		case ids, ok := <-watcherForApplicationChan:
			if !ok {
				return errors.New("secret migration tasks watcher for application closed")
			}
			w.config.Logger.Criticalf("secret migration tasks for application %v received!!!", ids)
			if err := w.migrateSecrets(
				func(ownerTag names.Tag) bool {
					w.config.Logger.Criticalf("migrateSecrets ownerTag %q, names.ApplicationTagKind?", ownerTag)
					return ownerTag.Kind() == names.ApplicationTagKind
				},
			); err != nil {
				return errors.Trace(err)
			}
		}
	}
}

// func (w *Worker) activeBackend() (jujusecrets.BackendsClient, error) {
// 	// TODO: cache ActiveID and only update it if model config(secret-backend) had change.
// 	cfg, err := w.config.SecretManagerFacade.GetSecretBackendConfig()
// 	if err != nil {
// 		return nil, errors.Trace(err)
// 	}
// 	backends, err := w.secretsBackends()
// 	if err != nil {
// 		return nil, errors.Trace(err)
// 	}
// 	activeBackend, err := backends.ForBackend(cfg.ActiveID)
// 	if err != nil {
// 		return nil, errors.Trace(err)
// 	}
// 	return activeBackend, nil
// }

func (w *Worker) migrateSecrets(filter func(tag names.Tag) bool) error {
	info, err := w.config.SecretManagerFacade.SecretMetadata()
	if err != nil {
		return errors.Trace(err)
	}
	w.config.Logger.Criticalf("migrateSecrets info: %s", pretty.Sprint(info))

	// TODO: cache ActiveID and only update it if model config(secret-backend) had change.
	cfg, err := w.config.SecretManagerFacade.GetSecretBackendConfig()
	if err != nil {
		return errors.Trace(err)
	}
	w.config.Logger.Criticalf("migrateSecrets cfg: %s", pretty.Sprint(cfg))
	activeBackendID := cfg.ActiveID

	backends, err := w.secretsBackends()
	if err != nil {
		return errors.Trace(err)
	}
	activeBackend, err := backends.ForBackend(activeBackendID)
	if err != nil {
		return errors.Trace(err)
	}

	for _, md := range info {
		w.config.Logger.Criticalf("migrateSecrets processing md: %s", pretty.Sprint(md))
		ownerTag, err := names.ParseTag(md.Metadata.OwnerTag)
		if err != nil {
			return errors.Trace(err)
		}
		if !filter(ownerTag) {
			w.config.Logger.Criticalf("migrateSecrets skipping md: %s", pretty.Sprint(md))
			continue
		}
		for _, revision := range md.Revisions {
			w.config.Logger.Criticalf("migrateSecrets processing revision: %s", pretty.Sprint(revision))
			content, err := w.config.SecretManagerFacade.GetRevisionContentInfo(md.Metadata.URI, revision, false)
			if err != nil {
				return errors.Trace(err)
			}
			w.config.Logger.Criticalf("migrateSecrets content: %s", pretty.Sprint(content))
			if err = content.Validate(); err != nil {
				return errors.Trace(err)
			}
			if content.ValueRef == nil {
				// ??????
				continue
			}
			w.config.Logger.Criticalf("migrateSecrets content.ValueRef.BackendID: %q, activeBackendID %q", content.ValueRef.BackendID, activeBackendID)
			if content.ValueRef.BackendID == activeBackendID {
				continue
			}

			oldBackend, err := backends.ForBackend(content.ValueRef.BackendID)
			if err != nil {
				return errors.Trace(err)
			}
			secretVal, err := oldBackend.GetContent(md.Metadata.URI, "", false, false)
			w.config.Logger.Criticalf("migrateSecrets secretVal: %s, err %#v", pretty.Sprint(secretVal), err)
			if err != nil {
				return errors.Trace(err)
			}
			_, err = activeBackend.SaveContent(md.Metadata.URI, revision, secretVal)
			w.config.Logger.Criticalf("migrateSecrets activeBackend.SaveContent(%q, %d, secretVal) err %#v", md.Metadata.URI, revision, err)
			if err != nil {
				// TODO: update TASK to FAILED!!!
				return errors.Trace(err)
			}
			// TODO: update TASK to COMPLETE!!!
		}
	}
	return nil
}

func appTag(tag names.UnitTag) names.ApplicationTag {
	appName, _ := names.UnitApplication(tag.Id())
	return names.NewApplicationTag(appName)
}
