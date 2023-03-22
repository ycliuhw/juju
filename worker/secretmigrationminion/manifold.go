// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretmigrationminion

// import (
// 	"github.com/juju/clock"
// 	"github.com/juju/errors"
// 	"github.com/juju/worker/v3"
// 	"github.com/juju/worker/v3/dependency"

// 	"github.com/juju/juju/api/agent/secretsmanager"
// 	"github.com/juju/juju/api/base"
// )

// // ManifoldConfig describes the resources used by the secretmigrationminion worker.
// type ManifoldConfig struct {
// 	APICallerName string
// 	Logger        Logger
// 	Clock         clock.Clock

// 	NewSecretManagerClient func(base.APICaller) (SecretManagerFacade, error)
// 	NewWorker              func(Config) (worker.Worker, error)
// }

// // NewSecretManagerClient returns a new secretsmanager facade.
// func NewSecretManagerClient(caller base.APICaller) SecretManagerFacade {
// 	return secretsmanager.NewClient(caller)
// }

// // Manifold returns a Manifold that encapsulates the secretmigrationminion worker.
// func Manifold(config ManifoldConfig) dependency.Manifold {
// 	return dependency.Manifold{
// 		Inputs: []string{
// 			config.APICallerName,
// 		},
// 		Start: config.start,
// 	}
// }

// // Validate is called by start to check for bad configuration.
// func (cfg ManifoldConfig) Validate() error {
// 	if cfg.APICallerName == "" {
// 		return errors.NotValidf("empty APICallerName")
// 	}
// 	if cfg.Logger == nil {
// 		return errors.NotValidf("nil Logger")
// 	}
// 	if cfg.Clock == nil {
// 		return errors.NotValidf("nil Clock")
// 	}
// 	return nil
// }

// // start is a StartFunc for a Worker manifold.
// func (cfg ManifoldConfig) start(context dependency.Context) (worker.Worker, error) {
// 	if err := cfg.Validate(); err != nil {
// 		return nil, errors.Trace(err)
// 	}

// 	var apiCaller base.APICaller
// 	if err := context.Get(cfg.APICallerName, &apiCaller); err != nil {
// 		return nil, errors.Trace(err)
// 	}

// 	facade, err := cfg.NewSecretManagerClient(apiCaller)
// 	if err != nil {
// 		return nil, errors.Trace(err)
// 	}

// 	worker, err := cfg.NewWorker(Config{
// 		SecretManagerFacade: facade,
// 		Logger:              cfg.Logger,
// 		Clock:               cfg.Clock,
// 	})
// 	if err != nil {
// 		return nil, errors.Trace(err)
// 	}
// 	return worker, nil
// }
