// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package secretusersupplied

import (
	"github.com/juju/errors"

	"github.com/juju/juju/api/base"
	apiwatcher "github.com/juju/juju/api/watcher"
	"github.com/juju/juju/core/secrets"
	"github.com/juju/juju/core/watcher"
	"github.com/juju/juju/rpc/params"
)

// Client is the api client for the SecretUserSuppliedManager facade.
type Client struct {
	facade base.FacadeCaller
}

// NewClient creates a secret backends manager api client.
func NewClient(caller base.APICaller) *Client {
	return &Client{
		facade: base.NewFacadeCaller(caller, "SecretUserSuppliedManager"),
	}
}

// WatchObsoleteRevisionsNeedPrune returns a watcher that triggers on secret
// obsolete revision changes changes.
func (c *Client) WatchObsoleteRevisionsNeedPrune() (watcher.StringsWatcher, error) {
	var result params.StringsWatchResult
	err := c.facade.FacadeCall("WatchObsoleteRevisionsNeedPrune", nil, &result)
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, params.TranslateWellKnownError(result.Error)
	}
	w := apiwatcher.NewStringsWatcher(c.facade.RawAPICaller(), result)
	return w, nil
}

func (c *Client) DeleteRevisions(uri *secrets.URI, revisions ...int) error {
	if len(revisions) == 0 {
		return errors.Errorf("at least one revision must be specified")
	}
	arg := params.DeleteSecretArg{
		URI:       uri.String(),
		Revisions: revisions,
	}

	var results params.ErrorResults
	err := c.facade.FacadeCall("DeleteRevisions", params.DeleteSecretArgs{Args: []params.DeleteSecretArg{arg}}, &results)
	if err != nil {
		return errors.Trace(err)
	}
	if len(results.Results) == 0 {
		return nil
	}
	if len(results.Results) != 1 {
		return errors.Errorf("unexpected number of results: %d", len(results.Results))
	}
	result := results.Results[0]
	if result.Error != nil {
		return params.TranslateWellKnownError(result.Error)
	}
	return nil
}