// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package crossmodelsecrets

import (
	"context"
	"reflect"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/names/v5"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/apiserver/common/crossmodel"
	"github.com/juju/juju/apiserver/common/secrets"
	"github.com/juju/juju/apiserver/facade"
	corelogger "github.com/juju/juju/core/logger"
	coremodel "github.com/juju/juju/core/model"
	"github.com/juju/juju/domain/credential"
	"github.com/juju/juju/internal/secrets/provider"
	"github.com/juju/juju/state"
)

// Register is called to expose a package of facades onto a given registry.
func Register(registry facade.FacadeRegistry) {
	registry.MustRegister("CrossModelSecrets", 1, func(stdCtx context.Context, ctx facade.ModelContext) (facade.Facade, error) {
		return newStateCrossModelSecretsAPI(stdCtx, ctx)
	}, reflect.TypeOf((*CrossModelSecretsAPI)(nil)))
}

// newStateCrossModelSecretsAPI creates a new server-side CrossModelSecrets API facade
// backed by global state.
func newStateCrossModelSecretsAPI(stdCtx context.Context, ctx facade.ModelContext) (*CrossModelSecretsAPI, error) {
	authCtxt := ctx.Resources().Get("offerAccessAuthContext").(*common.ValueResource).Value

	leadershipChecker, err := ctx.LeadershipChecker()
	if err != nil {
		return nil, errors.Trace(err)
	}
	model, err := ctx.State().Model()
	if err != nil {
		return nil, errors.Trace(err)
	}

	secretBackendConfigGetter := func(
		stdCtx context.Context, modelUUID string, sameController bool,
		adminModelCfg provider.ModelBackendConfigInfo,
		backendID string, consumer names.Tag,
	) (*provider.ModelBackendConfigInfo, error) {
		model, closer, err := ctx.StatePool().GetModel(modelUUID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		defer closer.Release()
		return secrets.BackendConfigInfo(stdCtx, secrets.SecretsModel(model), sameController, adminModelCfg, []string{backendID}, false, consumer, leadershipChecker)
	}
	serviceFactory := ctx.ServiceFactory()
	secretBackendAdminConfigGetter := func(stdCtx context.Context) (*provider.ModelBackendConfigInfo, error) {
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
		// return secrets.AdminBackendConfigInfo(stdCtx, secrets.SecretsModel(model), cloudService, credentialSerivce)
	}
	secretInfoGetter := func(modelUUID string) (SecretsState, SecretsConsumer, func() bool, error) {
		st, err := ctx.StatePool().Get(modelUUID)
		if err != nil {
			return nil, nil, nil, errors.Trace(err)
		}
		return state.NewSecrets(st.State), st, st.Release, nil
	}

	st := ctx.State()
	return NewCrossModelSecretsAPI(
		ctx.Resources(),
		authCtxt.(*crossmodel.AuthContext),
		st.ControllerUUID(),
		st.ModelUUID(),
		secretInfoGetter,
		secretBackendConfigGetter,
		secretBackendAdminConfigGetter,
		&crossModelShim{st.RemoteEntities()},
		&stateBackendShim{st},
		ctx.Logger().ChildWithTags("crossmodelsecrets", corelogger.SECRETS),
	)
}
