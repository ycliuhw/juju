// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretmigrationmanager

import (
	// "fmt"
	"time"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/names/v4"
	"github.com/juju/worker/v3"
	"github.com/juju/worker/v3/catacomb"

	// "github.com/juju/juju/core/secrets"
	"github.com/juju/juju/core/watcher"
	"github.com/juju/juju/environs/config"
	// "github.com/juju/juju/worker/fortress"
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

// Facade instances provide a set of API for the worker to deal with secret migration task changes.
type Facade interface {
	WatchForModelConfigChanges() (watcher.NotifyWatcher, error)
	ModelConfig() (*config.Config, error)

	WatchForFailedSecretMigrationTasks() (watcher.StringsWatcher, error)
	WatchForCompletedSecretMigrationTasks() (watcher.StringsWatcher, error)
	ScheduleSecretMigrationTasks() error
	RemoveSecretMigrationTasks() error
}

// Config defines the operation of the Worker.
type Config struct {
	ModelTag names.ModelTag
	Facade   Facade
	// Guard     fortress.Guard
	Logger Logger
	Clock  clock.Clock
}

// Validate returns an error if config cannot drive the Worker.
func (config Config) Validate() error {
	if config.ModelTag.Id() == "" {
		return errors.NotValidf("empty ModelTag")
	}
	if config.Facade == nil {
		return errors.NotValidf("nil Facade")
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

// NewWorker returns a secretmigrationmanager Worker backed by config, or an error.
func NewWorker(config Config) (worker.Worker, error) {
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

	secretBackend string
}

// TODO: we should do backend connection validation (ping the backend) when we change the secret backend in model config!!!
// juju model-config secret-backend=myothersecrets
// Or we should have a secret backend validator worker keeps ping the backend to make sure it's alive.
// The worker should make the backend as inactive if it's not alive.

// TODO: we should notify the secret owner (uniter or leader units) to migrate the secret to the new backend.
// Because Juju doesn't want to know the secret content at all!!!!

// TODO: user created secrets should be migrated on the controller becasue they donot have an owner unit!!

// Kill is defined on worker.Worker.
func (w *Worker) Kill() {
	w.catacomb.Kill(nil)
}

// Wait is part of the worker.Worker interface.
func (w *Worker) Wait() error {
	return w.catacomb.Wait()
}

func (w *Worker) hasSecretBackendChanged() (bool, error) {
	modelConfig, err := w.config.Facade.ModelConfig()
	if err != nil {
		return false, errors.Annotate(err, "cannot load model configuration")
	}
	secretBackend := modelConfig.SecretBackend()
	if secretBackend == w.secretBackend {
		return false, nil
	}
	w.secretBackend = secretBackend
	return true, nil
}

func (w *Worker) loop() (err error) {
	modelWatcher, err := w.config.Facade.WatchForModelConfigChanges()
	if err != nil {
		return errors.Trace(err)
	}
	if err := w.catacomb.Add(modelWatcher); err != nil {
		return errors.Trace(err)
	}

	failedTaskWatcher, err := w.config.Facade.WatchForFailedSecretMigrationTasks()
	if err != nil {
		return errors.Trace(err)
	}
	if err := w.catacomb.Add(failedTaskWatcher); err != nil {
		return errors.Trace(err)
	}

	completedTaskWatcher, err := w.config.Facade.WatchForCompletedSecretMigrationTasks()
	if err != nil {
		return errors.Trace(err)
	}
	if err := w.catacomb.Add(completedTaskWatcher); err != nil {
		return errors.Trace(err)
	}

	var timeToProcessFailedTasks <-chan time.Time
	for {
		select {
		case <-w.catacomb.Dying():
			return w.catacomb.ErrDying()
		case _, ok := <-modelWatcher.Changes():
			if !ok {
				return errors.New("model configuration watch closed")
			}
			changed, err := w.hasSecretBackendChanged()
			if err != nil {
				return errors.Trace(err)
			}
			if !changed {
				continue
			}
			w.config.Logger.Criticalf("secret backend has changed, now scheduling secret migration tasks!!!")
			if err := w.config.Facade.ScheduleSecretMigrationTasks(); err != nil {
				return errors.Trace(err)
			}
		case failed, ok := <-failedTaskWatcher.Changes():
			if !ok {
				return errors.New("failed secret migration task watch closed")
			}
			//
			w.config.Logger.Criticalf("failed secret migration tasks: %+v", failed)
			if timeToProcessFailedTasks == nil {
				timeToProcessFailedTasks = w.config.Clock.NewTimer(5 * time.Second).Chan()
			}
		case <-timeToProcessFailedTasks:
			w.config.Logger.Criticalf("time to process failed secret migration tasks!!!")
			// if err := w.config.Facade.ReScheduleSecretMigrationTasks(); err != nil {
			// 	return errors.Trace(err)
			// }
			// _ = timeToProcessFailedTasks.Stop()
			timeToProcessFailedTasks = nil

		case completed, ok := <-completedTaskWatcher.Changes():
			if !ok {
				return errors.New("completed secret migration task watch closed")
			}
			w.config.Logger.Criticalf("completed secret migration tasks: %+v", completed)
			if err := w.config.Facade.RemoveSecretMigrationTasks(); err != nil {
				return errors.Trace(err)
			}
		}
	}
}
