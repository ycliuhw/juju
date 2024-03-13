// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secrets

import (
	"context"

	"github.com/juju/collections/set"
	"github.com/juju/names/v5"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/cloud"
	"github.com/juju/juju/core/secrets"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/state"
)

// Model defines a subset of state model methods.
type Model interface {
	ControllerUUID() string
	CloudName() string
	CloudCredentialTag() (names.CloudCredentialTag, bool)
	Config() (*config.Config, error)
	UUID() string
	Name() string
	Type() state.ModelType
	State() *state.State

	ModelConfig(context.Context) (*config.Config, error)
	WatchForModelConfigChanges() state.NotifyWatcher
}

type StatePool interface {
	GetModel(modelUUID string) (common.Model, func() bool, error)
}

type SecretsBackendService interface {
	GetSecretBackend(context.Context, string) (*secrets.SecretBackend, error)
	ListSecretBackends() ([]secrets.SecretBackend, error)
}

// SecretsConsumer instances provide secret consumer apis.
type SecretsConsumer interface {
	SecretAccess(uri *secrets.URI, subject names.Tag) (secrets.SecretRole, error)
}

// SecretsState instances provide secret apis.
type SecretsState interface {
	ListModelSecrets(all bool) (map[string]set.Strings, error)
}

// SecretsMetaState instances provide secret metadata apis.
type SecretsMetaState interface {
	ListSecrets(state.SecretsFilter) ([]*secrets.SecretMetadata, error)
	ListSecretRevisions(uri *secrets.URI) ([]*secrets.SecretRevisionMetadata, error)
	SecretGrants(uri *secrets.URI, role secrets.SecretRole) ([]secrets.AccessInfo, error)
	ChangeSecretBackend(state.ChangeSecretBackendParams) error
}

// ListSecretsState instances provide secret metadata apis.
type ListSecretsState interface {
	ListSecrets(state.SecretsFilter) ([]*secrets.SecretMetadata, error)
}

// SecretsRemoveState instances provide secret removal apis.
type SecretsRemoveState interface {
	DeleteSecret(*secrets.URI, ...int) ([]secrets.ValueRef, error)
	GetSecret(*secrets.URI) (*secrets.SecretMetadata, error)
	GetSecretRevision(uri *secrets.URI, revision int) (*secrets.SecretRevisionMetadata, error)
	ListSecretRevisions(uri *secrets.URI) ([]*secrets.SecretRevisionMetadata, error)
	ListSecrets(state.SecretsFilter) ([]*secrets.SecretMetadata, error)
}

// Credential represents a cloud credential.
type Credential interface {
	AuthType() cloud.AuthType
	Attributes() map[string]string
}

// SecretsModel wraps a state Model.
func SecretsModel(m *state.Model) Model {
	return &modelShim{m}
}

type modelShim struct {
	*state.Model
}

type SecretsGetter interface {
	GetSecret(*secrets.URI) (*secrets.SecretMetadata, error)
	GetSecretValue(*secrets.URI, int) (secrets.SecretValue, *secrets.ValueRef, error)
}
