// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretsmanager

import (
	"time"

	"github.com/juju/names/v4"
	"gopkg.in/macaroon.v2"

	"github.com/juju/juju/core/secrets"
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
	ListSecrets(state.SecretsFilter) ([]*secrets.SecretMetadata, error)
	ListSecretRevisions(uri *secrets.URI) ([]*secrets.SecretRevisionMetadata, error)
	WatchObsolete(owners []names.Tag) (state.StringsWatcher, error)
}

type CrossModelState interface {
	GetToken(entity names.Tag) (string, error)
	GetRemoteEntity(token string) (names.Tag, error)
	GetMacaroon(entity names.Tag) (*macaroon.Macaroon, error)
}

// The secret migration stuff should probably be moved to a new facade later.
type State interface {
	Application(string) (Application, error)
	Unit(name string) (Unit, error)
}

type Application interface {
	WatchSecretMigrationTasks() state.StringsWatcher
}

type Unit interface {
	WatchSecretMigrationTasks() state.StringsWatcher
}

type stateShim struct {
	*state.State
}

func (s stateShim) Application(id string) (Application, error) {
	app, err := s.State.Application(id)
	if err != nil {
		return nil, err
	}
	return applicationShim{app}, nil
}

func (s stateShim) Unit(name string) (Unit, error) {
	u, err := s.State.Unit(name)
	if err != nil {
		return nil, err
	}
	return unitShim{u}, nil
}

type applicationShim struct {
	*state.Application
}

func (a applicationShim) WatchSecretMigrationTasks() state.StringsWatcher {
	return a.Application.WatchSecretMigrationTasks()
}

type unitShim struct {
	*state.Unit
}

func (u unitShim) WatchSecretMigrationTasks() state.StringsWatcher {
	return u.Unit.WatchSecretMigrationTasks()
}

type SecretMigrationTasksWatcherAPI interface {
	WatchSecretMigrationTasks() state.StringsWatcher
}
