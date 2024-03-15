// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretbackends

import (
	"context"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/names/v5"

	apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/apiserver/facade"
	coremodel "github.com/juju/juju/core/model"
	"github.com/juju/juju/core/permission"
	"github.com/juju/juju/core/secrets"
	"github.com/juju/juju/domain/credential"
	"github.com/juju/juju/domain/secretbackend"
	"github.com/juju/juju/internal/secrets/provider"
	_ "github.com/juju/juju/internal/secrets/provider/all"
	"github.com/juju/juju/internal/secrets/provider/juju"
	"github.com/juju/juju/internal/uuid"
	"github.com/juju/juju/rpc/params"
)

// SecretBackendsAPI is the server implementation for the SecretBackends facade.
type SecretBackendsAPI struct {
	authorizer     facade.Authorizer
	controllerUUID string

	clock             clock.Clock
	backendService    SecretsBackendService
	cloudService      CloudService
	credentialService CredentialService
	modelService      ModelService
	model             Model
	// secretState    SecretsState
	statePool StatePool
}

func (s *SecretBackendsAPI) checkCanAdmin() error {
	return s.authorizer.HasPermission(permission.SuperuserAccess, names.NewControllerTag(s.controllerUUID))
}

// AddSecretBackends adds new secret backends.
func (s *SecretBackendsAPI) AddSecretBackends(ctx context.Context, args params.AddSecretBackendArgs) (params.ErrorResults, error) {
	result := params.ErrorResults{
		Results: make([]params.ErrorResult, len(args.Args)),
	}
	if err := s.checkCanAdmin(); err != nil {
		return result, errors.Trace(err)
	}
	for i, arg := range args.Args {
		if arg.ID == "" {
			uuid, err := uuid.NewUUID()
			if err != nil {
				result.Results[i].Error = apiservererrors.ServerError(err)
				continue
			}
			arg.ID = uuid.String()
		}
		err := s.backendService.CreateSecretBackend(ctx, secrets.SecretBackend{
			ID:                  arg.ID,
			Name:                arg.Name,
			BackendType:         arg.BackendType,
			TokenRotateInterval: arg.TokenRotateInterval,
			Config:              arg.Config,
		})
		result.Results[i].Error = apiservererrors.ServerError(err)
	}
	return result, nil
}

// func (s *SecretBackendsAPI) createBackend(ctx context.Context, id string, arg params.SecretBackend) error {
// 	if arg.Name == "" {
// 		return errors.NotValidf("missing backend name")
// 	}
// 	if arg.Name == juju.BackendName || arg.Name == provider.Auto {
// 		return errors.NotValidf("backend %q")
// 	}
// 	p, err := provider.Provider(arg.BackendType)
// 	if err != nil {
// 		return errors.Annotatef(err, "creating backend provider type %q", arg.BackendType)
// 	}
// 	configValidator, ok := p.(provider.ProviderConfig)
// 	if ok {
// 		defaults := configValidator.ConfigDefaults()
// 		if arg.Config == nil && len(defaults) > 0 {
// 			arg.Config = make(map[string]interface{})
// 		}
// 		for k, v := range defaults {
// 			if _, ok := arg.Config[k]; !ok {
// 				arg.Config[k] = v
// 			}
// 		}
// 		err = configValidator.ValidateConfig(nil, arg.Config)
// 		if err != nil {
// 			return errors.Annotatef(err, "invalid config for provider %q", arg.BackendType)
// 		}
// 	}
// 	if err := commonsecrets.PingBackend(p, arg.Config); err != nil {
// 		return errors.Trace(err)
// 	}

// 	var nextRotateTime *time.Time
// 	if arg.TokenRotateInterval != nil && *arg.TokenRotateInterval > 0 {
// 		if !provider.HasAuthRefresh(p) {
// 			return errors.NotSupportedf("token refresh on secret backend of type %q", p.Type())
// 		}
// 		nextRotateTime, err = secrets.NextBackendRotateTime(s.clock.Now(), *arg.TokenRotateInterval)
// 		if err != nil {
// 			return errors.Trace(err)
// 		}
// 	}
// 	err = s.backendService.CreateSecretBackend(ctx, secrets.SecretBackend{
// 		ID:                  id,
// 		Name:                arg.Name,
// 		BackendType:         arg.BackendType,
// 		TokenRotateInterval: arg.TokenRotateInterval,
// 		Config:              arg.Config,
// 	})
// 	if errors.Is(err, errors.AlreadyExists) {
// 		return errors.AlreadyExistsf("secret backend with ID %q", id)
// 	}
// 	return errors.Trace(err)
// }

// UpdateSecretBackends updates secret backends.
func (s *SecretBackendsAPI) UpdateSecretBackends(ctx context.Context, args params.UpdateSecretBackendArgs) (params.ErrorResults, error) {
	result := params.ErrorResults{
		Results: make([]params.ErrorResult, len(args.Args)),
	}
	if err := s.checkCanAdmin(); err != nil {
		return result, errors.Trace(err)
	}
	for i, arg := range args.Args {
		err := s.backendService.UpdateSecretBackend(ctx, secrets.SecretBackend{
			Name:                arg.Name,
			TokenRotateInterval: arg.TokenRotateInterval,
			Config:              arg.Config,
		}, arg.Force, arg.Reset...)
		result.Results[i].Error = apiservererrors.ServerError(err)
	}
	return result, nil
}

// func (s *SecretBackendsAPI) updateBackend(ctx context.Context, arg params.UpdateSecretBackendArg) error {
// 	if arg.Name == "" {
// 		return errors.NotValidf("missing backend name")
// 	}
// 	if arg.Name == juju.BackendName || arg.Name == provider.Auto {
// 		return errors.NotValidf("backend %q")
// 	}
// 	existing, err := s.backendService.GetSecretBackendByName(ctx, arg.Name)
// 	if err != nil {
// 		return errors.Trace(err)
// 	}
// 	p, err := provider.Provider(existing.BackendType)
// 	if err != nil {
// 		return errors.Trace(err)
// 	}

// 	cfg := make(map[string]interface{})
// 	for k, v := range existing.Config {
// 		cfg[k] = v
// 	}
// 	for k, v := range arg.Config {
// 		cfg[k] = v
// 	}
// 	for _, k := range arg.Reset {
// 		delete(cfg, k)
// 	}
// 	configValidator, ok := p.(provider.ProviderConfig)
// 	if ok {
// 		defaults := configValidator.ConfigDefaults()
// 		for _, k := range arg.Reset {
// 			if defaultVal, ok := defaults[k]; ok {
// 				cfg[k] = defaultVal
// 			}
// 		}
// 		err = configValidator.ValidateConfig(existing.Config, cfg)
// 		if err != nil {
// 			return errors.Annotatef(err, "invalid config for provider %q", existing.BackendType)
// 		}
// 	}
// 	if !arg.Force {
// 		if err := commonsecrets.PingBackend(p, cfg); err != nil {
// 			return errors.Trace(err)
// 		}
// 	}
// 	var nextRotateTime *time.Time
// 	if arg.TokenRotateInterval != nil && *arg.TokenRotateInterval > 0 {
// 		if !provider.HasAuthRefresh(p) {
// 			return errors.NotSupportedf("token refresh on secret backend of type %q", p.Type())
// 		}
// 		nextRotateTime, err = secrets.NextBackendRotateTime(s.clock.Now(), *arg.TokenRotateInterval)
// 		if err != nil {
// 			return errors.Trace(err)
// 		}
// 	}
// 	err = s.backendService.UpdateSecretBackend(ctx, secrets.SecretBackend{
// 		ID:                  existing.ID,
// 		Name:                arg.Name,
// 		TokenRotateInterval: arg.TokenRotateInterval,
// 		Config:              cfg,
// 	}, nextRotateTime)
// 	if errors.Is(err, errors.NotFound) {
// 		return errors.NotFoundf("secret backend %q", arg.Name)
// 	}
// 	return err
// }

// ListSecretBackends lists available secret backends.
func (s *SecretBackendsAPI) ListSecretBackends(ctx context.Context, arg params.ListSecretBackendsArgs) (params.ListSecretBackendsResults, error) {
	result := params.ListSecretBackendsResults{}
	if arg.Reveal {
		if err := s.checkCanAdmin(); err != nil {
			return result, errors.Trace(err)
		}
	}

	cld, err := s.cloudService.Cloud(ctx, s.model.CloudName())
	if err != nil {
		return result, errors.Trace(err)
	}
	tag, ok := s.model.CloudCredentialTag()
	if !ok {
		return result, errors.NotValidf("cloud credential for %s is empty", s.model.UUID())
	}
	cred, err := s.credentialService.CloudCredential(ctx, credential.IdFromTag(tag))
	if err != nil {
		return result, errors.Trace(err)
	}
	infos, err := s.backendService.BackendSummaryInfo(
		ctx, coremodel.UUID(s.model.UUID()), s.modelService, *cld, cred, arg.Reveal, secretbackend.SecretBackendFilter{
			Names: arg.Names, All: true,
		},
	)
	if err != nil {
		return result, err
	}
	for _, backend := range infos {
		result.Results = append(result.Results, params.SecretBackendResult{
			ID:         backend.ID,
			NumSecrets: backend.NumSecrets,
			Status:     backend.Status,
			Message:    backend.Message,
			Result: params.SecretBackend{
				Name:                backend.Name,
				BackendType:         backend.BackendType,
				TokenRotateInterval: backend.TokenRotateInterval,
				Config:              backend.Config,
			},
		})
	}
	return result, nil
}

// RemoveSecretBackends removes secret backends.
func (s *SecretBackendsAPI) RemoveSecretBackends(ctx context.Context, args params.RemoveSecretBackendArgs) (params.ErrorResults, error) {
	result := params.ErrorResults{
		Results: make([]params.ErrorResult, len(args.Args)),
	}
	if err := s.checkCanAdmin(); err != nil {
		return result, errors.Trace(err)
	}
	for i, arg := range args.Args {
		if arg.Name == "" {
			err := errors.NotValidf("missing backend name")
			result.Results[i].Error = apiservererrors.ServerError(err)
			continue
		}
		if arg.Name == juju.BackendName || arg.Name == provider.Auto {
			err := errors.NotValidf("backend %q", arg.Name)
			result.Results[i].Error = apiservererrors.ServerError(err)
			continue
		}
		backend, err := s.backendService.GetSecretBackendByName(ctx, arg.Name)
		if err != nil {
			result.Results[i].Error = apiservererrors.ServerError(err)
			continue
		}
		err = s.backendService.DeleteSecretBackend(ctx, backend.ID, arg.Force)
		result.Results[i].Error = apiservererrors.ServerError(err)
	}
	return result, nil
}

// func (s *SecretBackendsAPI) removeBackend(ctx context.Context, arg params.RemoveSecretBackendArg) error {
// 	if arg.Name == "" {
// 		return errors.NotValidf("missing backend name")
// 	}
// 	if arg.Name == juju.BackendName || arg.Name == provider.Auto {
// 		return errors.NotValidf("backend %q")
// 	}
// 	return s.backendService.DeleteSecretBackend(ctx, arg.Name, arg.Force)
// }
