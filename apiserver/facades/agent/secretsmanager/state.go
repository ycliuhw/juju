// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretsmanager

import (
	"context"
	"time"

	"github.com/juju/names/v5"
	"gopkg.in/macaroon.v2"

	coremodel "github.com/juju/juju/core/model"
	"github.com/juju/juju/core/secrets"
	"github.com/juju/juju/domain/model"
	"github.com/juju/juju/state"
)

// SecretTriggers instances provide secret rotation/expiry apis.
type SecretTriggers interface {
	WatchSecretsRotationChanges(owners []names.Tag) (state.SecretsTriggerWatcher, error)
	WatchSecretRevisionsExpiryChanges(owners []names.Tag) (state.SecretsTriggerWatcher, error)
	SecretRotated(uri *secrets.URI, next time.Time) error
}

// SecretsConsumer instances provide secret consumer apis.
type SecretsConsumer interface {
	GetSecretConsumer(*secrets.URI, names.Tag) (*secrets.SecretConsumerMetadata, error)
	GetURIByConsumerLabel(string, names.Tag) (*secrets.URI, error)
	SaveSecretConsumer(*secrets.URI, names.Tag, *secrets.SecretConsumerMetadata) error
	WatchConsumedSecretsChanges(consumer names.Tag) (state.StringsWatcher, error)
	GrantSecretAccess(*secrets.URI, state.SecretAccessParams) error
	RevokeSecretAccess(*secrets.URI, state.SecretAccessParams) error
	SecretAccess(uri *secrets.URI, subject names.Tag) (secrets.SecretRole, error)
}

type SecretsState interface {
	CreateSecret(*secrets.URI, state.CreateSecretParams) (*secrets.SecretMetadata, error)
	UpdateSecret(*secrets.URI, state.UpdateSecretParams) (*secrets.SecretMetadata, error)
	DeleteSecret(*secrets.URI, ...int) ([]secrets.ValueRef, error)
	GetSecret(*secrets.URI) (*secrets.SecretMetadata, error)
	GetSecretValue(*secrets.URI, int) (secrets.SecretValue, *secrets.ValueRef, error)
	GetSecretRevision(uri *secrets.URI, revision int) (*secrets.SecretRevisionMetadata, error)
	ListSecrets(state.SecretsFilter) ([]*secrets.SecretMetadata, error)
	ListSecretRevisions(uri *secrets.URI) ([]*secrets.SecretRevisionMetadata, error)
	WatchObsolete(owners []names.Tag) (state.StringsWatcher, error)
	ChangeSecretBackend(state.ChangeSecretBackendParams) error
	SecretGrants(uri *secrets.URI, role secrets.SecretRole) ([]secrets.AccessInfo, error)
}

type CrossModelState interface {
	GetToken(entity names.Tag) (string, error)
	GetRemoteEntity(token string) (names.Tag, error)
	GetMacaroon(entity names.Tag) (*macaroon.Macaroon, error)
}

// // SecretBackendService is an interface for interacting with secret backend service.
// type SecretBackendService interface {
// 	// GetSecretBackendConfigLegacy(ctx context.Context) (*provider.ModelBackendConfigInfo, error)
// 	// GetSecretBackendConfig(ctx context.Context, backendID string) (*provider.ModelBackendConfigInfo, error)
// 	GetSecretBackendConfigForAdmin(
// 		context.Context, *config.Config, cloud.Cloud, cloud.Credential,
// 	) (*provider.ModelBackendConfigInfo, error)
// 	// GetSecretBackendConfigForDrain(context.Context, string) (*provider.ModelBackendConfigInfo, error)
// }

// ModelService provides methods for working with models for backend service.
type ModelService interface {
	GetModel(ctx context.Context, uuid coremodel.UUID) (*coremodel.Model, error)
	GetSecretBackend(ctx context.Context, modelUUID coremodel.UUID) (model.SecretBackendIdentifier, error)
}
