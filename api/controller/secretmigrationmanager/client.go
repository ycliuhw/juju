// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretmigrationmanager

import (
	"github.com/juju/errors"
	// "github.com/juju/names/v4"

	"github.com/juju/juju/api/base"
	"github.com/juju/juju/api/common"
	apiwatcher "github.com/juju/juju/api/watcher"
	apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/core/watcher"
	"github.com/juju/juju/rpc/params"
)

// Client provides access to the SecretMigrationManager API facade.
type Client struct {
	facade base.FacadeCaller
	*common.ModelWatcher
}

// NewClient creates a new client-side SecretMigrationManager API facade.
func NewClient(caller base.APICaller) (*Client, error) {
	_, isModel := caller.ModelTag()
	if !isModel {
		return nil, errors.New("expected model specific API connection")
	}
	facadeCaller := base.NewFacadeCaller(caller, "SecretMigrationManager")
	return &Client{
		facade:       facadeCaller,
		ModelWatcher: common.NewModelWatcher(facadeCaller),
	}, nil
}

// WatchForFailedSecretMigrationTasks returns a StringsWatcher that notifies of failed secret migration tasks.
func (c *Client) WatchForFailedSecretMigrationTasks() (watcher.StringsWatcher, error) {
	var result params.StringsWatchResult
	if err := c.facade.FacadeCall("WatchForFailedSecretMigrationTasks", nil, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, apiservererrors.RestoreError(result.Error)
	}
	w := apiwatcher.NewStringsWatcher(c.facade.RawAPICaller(), result)
	return w, nil
}

// WatchForCompletedSecretMigrationTasks returns a StringsWatcher that notifies of completed secret migration tasks.
func (c *Client) WatchForCompletedSecretMigrationTasks() (watcher.StringsWatcher, error) {
	var result params.StringsWatchResult
	if err := c.facade.FacadeCall("WatchForCompletedSecretMigrationTasks", nil, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, apiservererrors.RestoreError(result.Error)
	}
	w := apiwatcher.NewStringsWatcher(c.facade.RawAPICaller(), result)
	return w, nil
}

// ScheduleSecretMigrationTasks schedules secret migration tasks.
func (c *Client) ScheduleSecretMigrationTasks() error {
	err := c.facade.FacadeCall("ScheduleSecretMigrationTasks", nil, nil)
	return errors.Trace(err)
}

// RemoveSecretMigrationTasks removes the specified completed secret migration task.
func (c *Client) RemoveSecretMigrationTasks() error {
	err := c.facade.FacadeCall("RemoveSecretMigrationTasks", nil, nil)
	return errors.Trace(err)
}
