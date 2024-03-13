// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretbackends

import (
	"context"
	"reflect"

	"github.com/juju/clock"

	apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/apiserver/facade"
	"github.com/juju/juju/internal/secrets/provider"
)

// Register is called to expose a package of facades onto a given registry.
func Register(registry facade.FacadeRegistry) {
	registry.MustRegister("SecretBackends", 1, func(stdCtx context.Context, ctx facade.ModelContext) (facade.Facade, error) {
		return newSecretBackendsAPI(ctx)
	}, reflect.TypeOf((*SecretBackendsAPI)(nil)))
}

// newSecretBackendsAPI creates a SecretBackendsAPI.
func newSecretBackendsAPI(context facade.ModelContext) (*SecretBackendsAPI, error) {
	if !context.Auth().AuthClient() {
		return nil, apiservererrors.ErrPerm
	}
	model, err := context.State().Model()
	if err != nil {
		return nil, err
	}

	serviceFactory := context.ServiceFactory()
	return &SecretBackendsAPI{
		authorizer:        context.Auth(),
		controllerUUID:    context.State().ControllerUUID(),
		clock:             clock.WallClock,
		cloudService:      serviceFactory.Cloud(),
		credentialService: serviceFactory.Credential(),
		model:             model,
		modelService:      serviceFactory.Model(),
		backendService: serviceFactory.SecretBackend(
			clock.WallClock, model.ControllerUUID(), provider.Provider,
		),
		// secretState: state.NewSecrets(context.State()),
		statePool: &statePoolShim{pool: context.StatePool()},
	}, nil
}
