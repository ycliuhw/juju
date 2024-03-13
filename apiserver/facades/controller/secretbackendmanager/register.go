// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretbackendmanager

import (
	"context"
	"reflect"

	"github.com/juju/clock"
	"github.com/juju/errors"

	apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/apiserver/facade"
	"github.com/juju/juju/internal/secrets/provider"
)

// Register is called to expose a package of facades onto a given registry.
func Register(registry facade.FacadeRegistry) {
	registry.MustRegister("SecretBackendsManager", 1, func(stdCtx context.Context, ctx facade.ModelContext) (facade.Facade, error) {
		return NewSecretBackendsManagerAPI(ctx)
	}, reflect.TypeOf((*SecretBackendsManagerAPI)(nil)))
}

// NewSecretBackendsManagerAPI creates a SecretBackendsManagerAPI.
func NewSecretBackendsManagerAPI(context facade.ModelContext) (*SecretBackendsManagerAPI, error) {
	if !context.Auth().AuthController() {
		return nil, apiservererrors.ErrPerm
	}
	model, err := context.State().Model()
	if err != nil {
		return nil, errors.Trace(err)
	}
	sf := context.ServiceFactory()
	return &SecretBackendsManagerAPI{
		watcherRegistry: context.WatcherRegistry(),
		controllerUUID:  model.ControllerUUID(),
		modelUUID:       model.UUID(),
		modelName:       model.Name(),
		backendService: sf.SecretBackend(
			clock.WallClock, model.ControllerUUID(), provider.Provider,
		),
		clock:  clock.WallClock,
		logger: context.Logger().Child("secretbackendmanager"),
	}, nil
}
