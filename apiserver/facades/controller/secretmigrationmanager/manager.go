// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretmigrationmanager

import (
	// "github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/names/v4"

	"github.com/juju/juju/apiserver/common"
	apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/apiserver/facade"
	"github.com/juju/juju/rpc/params"
	"github.com/juju/juju/state/watcher"
)

var logger = loggo.GetLogger("juju.apiserver.secretmigrationmanager")

// Facade provides access to the CAASFirewaller API facade for sidecar applications.
type Facade struct {
	// *common.LifeGetter
	// *common.AgentEntityWatcher

	*common.ModelWatcher

	resources   facade.Resources
	accessModel common.GetAuthFunc
	st          State
}

func newFacade(
	st State,
	resources facade.Resources,
	authorizer facade.Authorizer,
) (*Facade, error) {
	if !authorizer.AuthController() {
		return nil, apiservererrors.ErrPerm
	}

	// ModelConfig() and WatchForModelConfigChanges() are allowed
	// with unrestricted access.
	modelWatcher := common.NewModelWatcher(
		st,
		resources,
		authorizer,
	)

	return &Facade{
		accessModel: common.AuthFuncForTagKind(names.ModelTagKind),

		ModelWatcher: modelWatcher,
		// LifeGetter: common.NewLifeGetter(
		// 	st, common.AuthAny(
		// 		common.AuthFuncForTagKind(names.ApplicationTagKind),
		// 		common.AuthFuncForTagKind(names.UnitTagKind),
		// 	),
		// ),
		// AgentEntityWatcher: common.NewAgentEntityWatcher(
		// 	st,
		// 	resources,
		// 	accessApplication,
		// ),
		resources: resources,
		st:        st,
	}, nil

}

// WatchForFailedSecretMigrationTasks returns a StringsWatcher that notifies of failed secret migration tasks.
func (f *Facade) WatchForFailedSecretMigrationTasks() (params.StringsWatchResult, error) {
	watch := f.st.WatchForFailedSecretMigrationTasks()
	// Consume the initial event and forward it to the result.
	if changes, ok := <-watch.Changes(); ok {
		return params.StringsWatchResult{
			StringsWatcherId: f.resources.Register(watch),
			Changes:          changes,
		}, nil
	}
	return params.StringsWatchResult{}, watcher.EnsureErr(watch)
}

// WatchForCompletedSecretMigrationTasks returns a StringsWatcher that notifies of completed secret migration tasks.
func (f *Facade) WatchForCompletedSecretMigrationTasks() (params.StringsWatchResult, error) {
	watch := f.st.WatchForCompletedSecretMigrationTasks()
	// Consume the initial event and forward it to the result.
	if changes, ok := <-watch.Changes(); ok {
		return params.StringsWatchResult{
			StringsWatcherId: f.resources.Register(watch),
			Changes:          changes,
		}, nil
	}
	return params.StringsWatchResult{}, watcher.EnsureErr(watch)
}

// ScheduleSecretMigrationTasks schedules secret migration tasks.
func (f *Facade) ScheduleSecretMigrationTasks() error {
	return f.st.ScheduleSecretMigrationTasks()
}

// RemoveSecretMigrationTasks removes the completed secret migration tasks.
func (f *Facade) RemoveSecretMigrationTasks() error {
	return f.st.RemoveSecretMigrationTasks()
}
