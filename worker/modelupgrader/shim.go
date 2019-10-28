// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelupgrader

import (
	"gopkg.in/juju/names.v3"

	"github.com/juju/juju/api/base"
	"github.com/juju/juju/api/common/cloudspec"
	"github.com/juju/juju/api/modelupgrader"
	"github.com/juju/juju/environs"
)

func NewFacade(apiCaller base.APICaller) (Facade, error) {
	facade := modelupgrader.NewClient(apiCaller)
	return facade, nil
}

type CloudAPI interface {
	CloudSpec() (environs.CloudSpec, error)
}

func NewCloudSpecAPI(apiCaller base.APICaller, tag names.ModelTag) (Facade, error) {
	facade := cloudspec.NewCloudSpecAPI(apiCaller, tag)
	return facade, nil
}
