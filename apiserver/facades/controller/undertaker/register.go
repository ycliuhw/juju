// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package undertaker

import (
	"context"
	"reflect"

	"github.com/juju/clock"
	"github.com/juju/errors"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/apiserver/common/cloudspec"
	"github.com/juju/juju/apiserver/facade"
	coremodel "github.com/juju/juju/core/model"
	"github.com/juju/juju/domain/credential"
	"github.com/juju/juju/internal/secrets/provider"
)

// Register is called to expose a package of facades onto a given registry.
func Register(registry facade.FacadeRegistry) {
	registry.MustRegister("Undertaker", 1, func(stdCtx context.Context, ctx facade.ModelContext) (facade.Facade, error) {
		return newUndertakerFacade(ctx)
	}, reflect.TypeOf((*UndertakerAPI)(nil)))
}

// newUndertakerFacade creates a new instance of the undertaker API.
func newUndertakerFacade(ctx facade.ModelContext) (*UndertakerAPI, error) {
	st := ctx.State()
	model, err := st.Model()
	if err != nil {
		return nil, errors.Trace(err)
	}
	serviceFactory := ctx.ServiceFactory()
	cloudService := serviceFactory.Cloud()
	modelService := serviceFactory.Model()
	credentialSerivce := serviceFactory.Credential()
	secretsBackendsGetter := func(ctx context.Context) (*provider.ModelBackendConfigInfo, error) {
		backendService := serviceFactory.SecretBackend(
			clock.WallClock, model.ControllerUUID(), provider.Provider,
		)
		cld, err := cloudService.Cloud(ctx, model.CloudName())
		if err != nil {
			return nil, errors.Trace(err)
		}
		tag, ok := model.CloudCredentialTag()
		if !ok {
			return nil, errors.NotValidf("cloud credential for %s is empty", model.UUID())
		}
		cred, err := credentialSerivce.CloudCredential(ctx, credential.IdFromTag(tag))
		if err != nil {
			return nil, errors.Trace(err)
		}
		return backendService.GetSecretBackendConfigForAdmin(ctx, coremodel.UUID(model.UUID()), modelService, *cld, cred)
		// return secrets.AdminBackendConfigInfo(ctx, secrets.SecretsModel(model), cloudService, credentialService)
	}
	cloudSpecAPI := cloudspec.NewCloudSpec(
		ctx.Resources(),
		cloudspec.MakeCloudSpecGetterForModel(st, cloudService, credentialSerivce),
		cloudspec.MakeCloudSpecWatcherForModel(st, cloudService),
		cloudspec.MakeCloudSpecCredentialWatcherForModel(st),
		cloudspec.MakeCloudSpecCredentialContentWatcherForModel(st, ctx.ServiceFactory().Credential()),
		common.AuthFuncForTag(model.ModelTag()),
	)
	return newUndertakerAPI(&stateShim{st}, ctx.Resources(), ctx.Auth(), secretsBackendsGetter, cloudSpecAPI)
}
