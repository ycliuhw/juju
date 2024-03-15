// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretbackends

import (
	"context"

	"github.com/juju/collections/set"
	"github.com/juju/errors"
	"github.com/juju/names/v5"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/cloud"
	coremodel "github.com/juju/juju/core/model"
	"github.com/juju/juju/core/secrets"
	"github.com/juju/juju/domain/credential"
	"github.com/juju/juju/domain/model"
	"github.com/juju/juju/domain/secretbackend"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/state"
)

// SecretsBackendService is an interface for interacting with secret backend service.
type SecretsBackendService interface {
	CreateSecretBackend(context.Context, secrets.SecretBackend) error
	UpdateSecretBackend(context.Context, secrets.SecretBackend, bool, ...string) error
	DeleteSecretBackend(context.Context, string, bool) error
	GetSecretBackendByName(context.Context, string) (*secrets.SecretBackend, error)

	BackendSummaryInfo(
		ctx context.Context,
		modelUUID coremodel.UUID, model secretbackend.ModelGetter, cloud cloud.Cloud, cred cloud.Credential,
		reveal bool, filter secretbackend.SecretBackendFilter,
	) ([]*secretbackend.SecretBackendInfo, error)
}

// CloudService provides access to clouds.
type CloudService interface {
	Cloud(ctx context.Context, name string) (*cloud.Cloud, error)
}

// CredentialService exposes State methods needed by credential manager.
type CredentialService interface {
	CloudCredential(ctx context.Context, id credential.ID) (cloud.Credential, error)
}

// ModelService provides access to model information.
type ModelService interface {
	GetModel(ctx context.Context, uuid coremodel.UUID) (*coremodel.Model, error)
	GetSecretBackend(ctx context.Context, modelUUID coremodel.UUID) (model.SecretBackendIdentifier, error)
}

type SecretsState interface {
	ListModelSecrets(all bool) (map[string]set.Strings, error)
}

type StatePool interface {
	GetModel(modelUUID string) (common.Model, func() bool, error)
}

type statePoolShim struct {
	pool *state.StatePool
}

func (s *statePoolShim) GetModel(modelUUID string) (common.Model, func() bool, error) {
	m, hp, err := s.pool.GetModel(modelUUID)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return m, hp.Release, nil
}

type Model interface {
	UUID() string
	Config() (*config.Config, error)
	CloudName() string
	CloudCredentialTag() (names.CloudCredentialTag, bool)
}
