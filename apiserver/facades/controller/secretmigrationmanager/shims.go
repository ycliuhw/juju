// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretmigrationmanager

import (
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/state"
)

// State provides the subset of global state required by the
// remote secretmigrationmanager facade.
type State interface {
	state.ModelAccessor

	ScheduleSecretMigrationTasks() error
	RemoveSecretMigrationTasks() error
	WatchForFailedSecretMigrationTasks() state.StringsWatcher
	WatchForCompletedSecretMigrationTasks() state.StringsWatcher
}

type stateShim struct {
	st *state.State
	m  *state.Model
}

func (s stateShim) WatchForModelConfigChanges() state.NotifyWatcher {
	return s.m.WatchForModelConfigChanges()
}

func (s stateShim) ModelConfig() (*config.Config, error) {
	return s.m.ModelConfig()
}

func (s stateShim) ScheduleSecretMigrationTasks() error {
	return s.m.ScheduleSecretMigrationTasks()
}

func (s stateShim) RemoveSecretMigrationTasks() error {
	return s.m.RemoveSecretMigrationTasks()
}

func (s stateShim) WatchForFailedSecretMigrationTasks() state.StringsWatcher {
	return s.m.WatchForFailedSecretMigrationTasks()
}

func (s stateShim) WatchForCompletedSecretMigrationTasks() state.StringsWatcher {
	return s.m.WatchForCompletedSecretMigrationTasks()
}
