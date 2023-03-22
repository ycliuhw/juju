// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretmigrationmanager

import (
	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/worker/v3"
	"github.com/juju/worker/v3/dependency"

	"github.com/juju/juju/api/base"
	"github.com/juju/juju/api/controller/secretmigrationmanager"
)

// ManifoldConfig describes the resources used by the secretmigrationmanager worker.
type ManifoldConfig struct {
	APICallerName string
	Logger        Logger
	Clock         clock.Clock

	NewFacade func(base.APICaller) (Facade, error)
	NewWorker func(Config) (worker.Worker, error)
}

// NewClient returns a new secretmigrationmanager facade.
func NewClient(caller base.APICaller) (Facade, error) {
	return secretmigrationmanager.NewClient(caller)
}

// Manifold returns a Manifold that encapsulates the secretmigrationmanager worker.
func Manifold(config ManifoldConfig) dependency.Manifold {
	return dependency.Manifold{
		Inputs: []string{
			config.APICallerName,
		},
		Start: config.start,
	}
}

// Validate is called by start to check for bad configuration.
func (cfg ManifoldConfig) Validate() error {
	if cfg.APICallerName == "" {
		return errors.NotValidf("empty APICallerName")
	}
	if cfg.Logger == nil {
		return errors.NotValidf("nil Logger")
	}
	if cfg.Clock == nil {
		return errors.NotValidf("nil Clock")
	}
	return nil
}

// start is a StartFunc for a Worker manifold.
func (cfg ManifoldConfig) start(context dependency.Context) (worker.Worker, error) {
	if err := cfg.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	var apiCaller base.APICaller
	if err := context.Get(cfg.APICallerName, &apiCaller); err != nil {
		return nil, errors.Trace(err)
	}
	modelTag, ok := apiCaller.ModelTag()
	if !ok {
		return nil, errors.New("API connection is controller-only (should never happen)")
	}
	facade, err := cfg.NewFacade(apiCaller)
	if err != nil {
		return nil, errors.Trace(err)
	}

	worker, err := cfg.NewWorker(Config{
		ModelTag: modelTag,
		Facade:   facade,
		Logger:   cfg.Logger,
		Clock:    cfg.Clock,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return worker, nil
}
