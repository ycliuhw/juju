// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package usersecretsdrain

import (
	stdcontext "context"
	"reflect"

	"github.com/juju/clock"
	"github.com/juju/errors"

	commonsecrets "github.com/juju/juju/apiserver/common/secrets"
	apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/apiserver/facade"
	"github.com/juju/juju/domain/credential"
	domainmodel "github.com/juju/juju/domain/model"
	"github.com/juju/juju/internal/secrets/provider"
	"github.com/juju/juju/state"
)

// Register is called to expose a package of facades onto a given registry.
func Register(registry facade.FacadeRegistry) {
	registry.MustRegister("UserSecretsDrain", 1, func(stdCtx stdcontext.Context, ctx facade.ModelContext) (facade.Facade, error) {
		return newUserSecretsDrainAPI(ctx)
	}, reflect.TypeOf((*SecretsDrainAPI)(nil)))
}

// newUserSecretsDrainAPI creates a SecretsDrainAPI for draining user secrets.
func newUserSecretsDrainAPI(context facade.ModelContext) (*SecretsDrainAPI, error) {
	if !context.Auth().AuthController() {
		return nil, apiservererrors.ErrPerm
	}
	leadershipChecker, err := context.LeadershipChecker()
	if err != nil {
		return nil, errors.Trace(err)
	}
	model, err := context.State().Model()
	if err != nil {
		return nil, errors.Trace(err)
	}
	serviceFactory := context.ServiceFactory()

	authTag := model.ModelTag()
	commonDrainAPI, err := commonsecrets.NewSecretsDrainAPI(
		authTag,
		context.Auth(),
		context.Logger().Child("usersecretsdrain"),
		leadershipChecker,
		commonsecrets.SecretsModel(model),
		state.NewSecrets(context.State()),
		context.State(),
		context.WatcherRegistry(),
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	secretBackendAdminConfigGetter := func(stdCtx stdcontext.Context) (*provider.ModelBackendConfigInfo, error) {
		cloudService := serviceFactory.Cloud()
		credentialSerivce := serviceFactory.Credential()
		modelService := serviceFactory.Model()
		backendService := serviceFactory.SecretBackend(
			clock.WallClock, model.ControllerUUID(), provider.Provider,
		)
		cld, err := cloudService.Get(stdCtx, model.CloudName())
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
		return backendService.GetSecretBackendConfigForAdmin(stdCtx, domainmodel.UUID(model.UUID()), modelService, *cld, cred)
		// return secrets.AdminBackendConfigInfo(stdCtx, secrets.SecretsModel(model), cloudService, credentialSerivce)
	}

	secretBackendConfigGetter := func(ctx stdcontext.Context, backendIDs []string, wantAll bool) (*provider.ModelBackendConfigInfo, error) {
		adminCfg, err := secretBackendAdminConfigGetter(ctx)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return commonsecrets.BackendConfigInfo(
			ctx, commonsecrets.SecretsModel(model), true, *adminCfg,
			backendIDs, wantAll, authTag, leadershipChecker,
		)
	}
	secretBackendDrainConfigGetter := func(ctx stdcontext.Context, backendID string) (*provider.ModelBackendConfigInfo, error) {
		adminCfg, err := secretBackendAdminConfigGetter(ctx)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return commonsecrets.DrainBackendConfigInfo(
			ctx, backendID, commonsecrets.SecretsModel(model),
			*adminCfg,
			authTag, leadershipChecker,
		)
	}

	return &SecretsDrainAPI{
		SecretsDrainAPI:     commonDrainAPI,
		drainConfigGetter:   secretBackendDrainConfigGetter,
		backendConfigGetter: secretBackendConfigGetter,
		secretsState:        state.NewSecrets(context.State()),
	}, nil
}
