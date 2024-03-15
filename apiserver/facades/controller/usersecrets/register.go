// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package usersecrets

import (
	stdcontext "context"
	"reflect"

	"github.com/juju/clock"
	"github.com/juju/errors"

	apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/apiserver/facade"
	coremodel "github.com/juju/juju/core/model"
	"github.com/juju/juju/domain/credential"
	"github.com/juju/juju/internal/secrets/provider"
	"github.com/juju/juju/state"
)

// Register is called to expose a package of facades onto a given registry.
func Register(registry facade.FacadeRegistry) {
	registry.MustRegister("UserSecretsManager", 1, func(stdCtx stdcontext.Context, ctx facade.ModelContext) (facade.Facade, error) {
		return NewUserSecretsManager(ctx)
	}, reflect.TypeOf((*UserSecretsManager)(nil)))
}

// NewUserSecretsManager creates a UserSecretsManager.
func NewUserSecretsManager(context facade.ModelContext) (*UserSecretsManager, error) {
	if !context.Auth().AuthController() {
		return nil, apiservererrors.ErrPerm
	}
	model, err := context.State().Model()
	if err != nil {
		return nil, errors.Trace(err)
	}

	serviceFactory := context.ServiceFactory()
	backendConfigGetter := func(stdCtx stdcontext.Context) (*provider.ModelBackendConfigInfo, error) {
		cloudService := serviceFactory.Cloud()
		credentialSerivce := serviceFactory.Credential()
		modelService := serviceFactory.Model()
		backendService := serviceFactory.SecretBackend(
			clock.WallClock, model.ControllerUUID(), provider.Provider,
		)
		cld, err := cloudService.Cloud(stdCtx, model.CloudName())
		if err != nil {
			return nil, errors.Trace(err)
		}
		tag, ok := model.CloudCredentialTag()
		if !ok {
			return nil, errors.NotValidf("cloud credential for %s is empty", model.UUID())
		}
		cred, err := credentialSerivce.CloudCredential(stdCtx, credential.IdFromTag(tag))
		if err != nil {
			return nil, errors.Trace(err)
		}
		return backendService.GetSecretBackendConfigForAdmin(stdCtx, coremodel.UUID(model.UUID()), modelService, *cld, cred)
		// return secrets.AdminBackendConfigInfo(
		// 	ctx, secrets.SecretsModel(model),
		// 	serviceFactory.Cloud(), serviceFactory.Credential(),
		// )
	}

	return &UserSecretsManager{
		authorizer:          context.Auth(),
		resources:           context.Resources(),
		authTag:             context.Auth().GetAuthTag(),
		controllerUUID:      context.State().ControllerUUID(),
		modelUUID:           context.State().ModelUUID(),
		secretsState:        state.NewSecrets(context.State()),
		backendConfigGetter: backendConfigGetter,
	}, nil
}
