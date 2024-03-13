// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelconfig

import (
	"context"

	"github.com/juju/names/v5"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/core/constraints"
	coresecrets "github.com/juju/juju/core/secrets"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/state"
)

// Backend contains the state.State methods used in this package,
// allowing stubs to be created for testing.
type Backend interface {
	common.BlockGetter
	ControllerTag() names.ControllerTag
	ModelTag() names.ModelTag
	ModelConfigValues() (config.ConfigValues, error)
	UpdateModelConfig(map[string]interface{}, []string, ...state.ValidateConfigFunc) error
	Sequences() (map[string]int, error)
	SpaceByName(string) error
	SetModelConstraints(value constraints.Value) error
	ModelConstraints() (constraints.Value, error)
}

// SecretBackendService is an interface for interacting with secret backend service.
type SecretBackendService interface {
	GetSecretBackendByName(context.Context, string) (*coresecrets.SecretBackend, error)
	CheckSecretBackend(ctx context.Context, backendID string) error
}

type stateShim struct {
	*state.State
	model                    *state.Model
	configSchemaSourceGetter config.ConfigSchemaSourceGetter
}

func (st stateShim) UpdateModelConfig(u map[string]interface{}, r []string, a ...state.ValidateConfigFunc) error {
	return st.model.UpdateModelConfig(st.configSchemaSourceGetter, u, r, a...)
}

func (st stateShim) ModelConfigValues() (config.ConfigValues, error) {
	return st.model.ModelConfigValues(st.configSchemaSourceGetter)
}

func (st stateShim) ModelTag() names.ModelTag {
	m, err := st.State.Model()
	if err != nil {
		return names.NewModelTag(st.State.ModelUUID())
	}

	return m.ModelTag()
}

func (st stateShim) SpaceByName(name string) error {
	_, err := st.State.SpaceByName(name)
	return err
}

// NewStateBackend creates a backend for the facade to use.
func NewStateBackend(m *state.Model, configSchemaSourceGetter config.ConfigSchemaSourceGetter) Backend {
	return stateShim{
		State:                    m.State(),
		model:                    m,
		configSchemaSourceGetter: configSchemaSourceGetter,
	}
}
