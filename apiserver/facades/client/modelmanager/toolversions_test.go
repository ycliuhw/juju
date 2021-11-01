// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelmanager_test

import (
	"github.com/golang/mock/gomock"
	"github.com/juju/names/v4"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version/v2"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver/facades/client/modelmanager"
	"github.com/juju/juju/apiserver/facades/client/modelmanager/mocks"
	"github.com/juju/juju/apiserver/params"
	apiservertesting "github.com/juju/juju/apiserver/testing"
	coreos "github.com/juju/juju/core/os"
	"github.com/juju/juju/docker"
	"github.com/juju/juju/docker/registry"
	"github.com/juju/juju/docker/registry/image"
	registrymocks "github.com/juju/juju/docker/registry/mocks"
	jujustate "github.com/juju/juju/state"
	"github.com/juju/juju/tools"
	jujuversion "github.com/juju/juju/version"
)

func (s *modelManagerNewSuite) TestToolVersions(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	model := mocks.NewMockModel(ctrl)
	model.EXPECT().IsControllerModel().Return(true)
	model.EXPECT().Type().Return(jujustate.ModelTypeCAAS)

	state := mocks.NewMockState(ctrl)
	state.EXPECT().Model().Return(model, nil)
	state.EXPECT().Release()

	statePool := mocks.NewMockStatePool(ctrl)
	statePool.EXPECT().Get(s.st.model.tag.Id()).Return(state, nil)

	s.PatchValue(&jujuversion.Current, version.MustParse("2.9.2"))
	registryAPI := registrymocks.NewMockRegistry(ctrl)
	registryAPI.EXPECT().Tags("jujud-operator").Return(
		tools.Versions{
			image.NewImageInfo(version.MustParse("2.9.1")), // It will be ignored because it's older than `version.Current`.
			image.NewImageInfo(version.MustParse("2.9.10")),
			image.NewImageInfo(version.MustParse("2.9.11")),
		}, nil,
	)
	registryAPI.EXPECT().GetArchitecture("jujud-operator", "2.9.10").Return("amd64", nil)
	registryAPI.EXPECT().GetArchitecture("jujud-operator", "2.9.11").Return("amd64", nil)
	registryAPI.EXPECT().Close().Return(nil)

	api, err := modelmanager.NewModelManagerAPI(
		s.st, &mockState{}, statePool, nil, nil, s.authoriser, s.st.model, s.callContext,
		func(repoDetails docker.ImageRepoDetails) (registry.Registry, error) {
			c.Assert(repoDetails.Repository, jc.DeepEquals, `test-account`)
			c.Assert(repoDetails.ServerAddress, jc.DeepEquals, `quay.io`)
			c.Assert(repoDetails.Auth.String(), jc.DeepEquals, `xxxxx==`)
			return registryAPI, nil
		},
	)
	c.Assert(err, jc.ErrorIsNil)

	result, err := api.ToolVersions(params.Entity{
		Tag: s.st.model.tag.String(),
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Error, gc.IsNil)
	c.Assert(result.ToolVersions, gc.DeepEquals, tools.List{
		&tools.Tools{
			Version: version.Binary{
				Number:  version.MustParse("2.9.10"),
				Release: coreos.HostOSTypeName(),
				Arch:    "amd64",
			},
		},
		&tools.Tools{
			Version: version.Binary{
				Number:  version.MustParse("2.9.11"),
				Release: coreos.HostOSTypeName(),
				Arch:    "amd64",
			},
		},
	})
}

func (s *modelManagerNewSuite) TestToolVersionsDeniedForNonControllerModel(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	model := mocks.NewMockModel(ctrl)
	model.EXPECT().IsControllerModel().Return(false)

	state := mocks.NewMockState(ctrl)
	state.EXPECT().Model().Return(model, nil)
	state.EXPECT().Release()

	statePool := mocks.NewMockStatePool(ctrl)
	statePool.EXPECT().Get(s.st.model.tag.Id()).Return(state, nil)

	registryAPI := registrymocks.NewMockRegistry(ctrl)

	api, err := modelmanager.NewModelManagerAPI(
		s.st, &mockState{}, statePool, nil, nil, s.authoriser, s.st.model, s.callContext,
		func(repoDetails docker.ImageRepoDetails) (registry.Registry, error) {
			c.Assert(repoDetails.Repository, jc.DeepEquals, `test-account`)
			c.Assert(repoDetails.ServerAddress, jc.DeepEquals, `quay.io`)
			c.Assert(repoDetails.Auth.String(), jc.DeepEquals, `xxxxx==`)
			return registryAPI, nil
		},
	)
	c.Assert(err, jc.ErrorIsNil)

	_, err = api.ToolVersions(params.Entity{
		Tag: s.st.model.tag.String(),
	})
	c.Assert(err, gc.ErrorMatches, `model ".*" is not controller model`)
}

func (s *modelManagerNewSuite) TestToolVersionsDeniedForNonCaaSModel(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	model := mocks.NewMockModel(ctrl)
	model.EXPECT().IsControllerModel().Return(true)
	model.EXPECT().Type().Return(jujustate.ModelTypeIAAS)

	state := mocks.NewMockState(ctrl)
	state.EXPECT().Model().Return(model, nil)
	state.EXPECT().Release()

	statePool := mocks.NewMockStatePool(ctrl)
	statePool.EXPECT().Get(s.st.model.tag.Id()).Return(state, nil)

	registryAPI := registrymocks.NewMockRegistry(ctrl)

	api, err := modelmanager.NewModelManagerAPI(
		s.st, &mockState{}, statePool, nil, nil, s.authoriser, s.st.model, s.callContext,
		func(repoDetails docker.ImageRepoDetails) (registry.Registry, error) {
			c.Assert(repoDetails.Repository, jc.DeepEquals, `test-account`)
			c.Assert(repoDetails.ServerAddress, jc.DeepEquals, `quay.io`)
			c.Assert(repoDetails.Auth.String(), jc.DeepEquals, `xxxxx==`)
			return registryAPI, nil
		},
	)
	c.Assert(err, jc.ErrorIsNil)

	_, err = api.ToolVersions(params.Entity{
		Tag: s.st.model.tag.String(),
	})
	c.Assert(err, gc.ErrorMatches, `ToolVersions is for CAAS model only not implemented`)
}

func (s *modelManagerNewSuite) TestToolVersionsWithNoPermission(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	statePool := mocks.NewMockStatePool(ctrl)
	registryAPI := registrymocks.NewMockRegistry(ctrl)
	authoriser := apiservertesting.FakeAuthorizer{
		Tag: names.NewUserTag("user"),
	}

	api, err := modelmanager.NewModelManagerAPI(
		s.st, &mockState{}, statePool, nil, nil, authoriser, s.st.model, s.callContext,
		func(repoDetails docker.ImageRepoDetails) (registry.Registry, error) {
			c.Assert(repoDetails.Repository, jc.DeepEquals, `test-account`)
			c.Assert(repoDetails.ServerAddress, jc.DeepEquals, `quay.io`)
			c.Assert(repoDetails.Auth.String(), jc.DeepEquals, `xxxxx==`)
			return registryAPI, nil
		},
	)
	c.Assert(err, jc.ErrorIsNil)

	_, err = api.ToolVersions(params.Entity{
		Tag: s.st.model.tag.String(),
	})
	c.Assert(err, gc.ErrorMatches, `permission denied`)
}
