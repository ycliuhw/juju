// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ecs

import (
	"sync"

	"github.com/aws/aws-sdk-go/service/ecs"
	// _ "github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/juju/errors"

	// "github.com/juju/juju/environs"
	"github.com/juju/juju/cloud"
	cloudspec "github.com/juju/juju/environs/cloudspec"
	"github.com/juju/juju/environs/config"
)

type environ struct {
	name        string
	clusterName string

	cloud cloudspec.CloudSpec

	envCfgLock     sync.Mutex
	envCfgUnlocked *config.Config

	clientUnlocked *ecs.ECS
}

// var _ environs.Environ = (*environ)(nil)

//go:generate go run github.com/golang/mock/mockgen -package mocks -destination mocks/ecs_mock.go github.com/aws/aws-sdk-go/service/ecs/ecsiface ECSAPI
func new(clusterName string, cloud cloudspec.CloudSpec, envCfg *config.Config) (*environ, error) {
	env := &environ{
		name:           envCfg.Name(),
		clusterName:    clusterName,
		cloud:          cloud,
		envCfgUnlocked: envCfg,
	}
	var err error
	if env.clientUnlocked, err = newECSClient(cloud); err != nil {
		return nil, errors.Trace(err)
	}
	return env, nil
}

func validateCloudSpec(c cloudspec.CloudSpec) error {
	if err := c.Validate(); err != nil {
		return errors.Trace(err)
	}
	if c.Credential == nil {
		return errors.NotValidf("missing credential")
	}
	if authType := c.Credential.AuthType(); authType != cloud.AccessKeyAuthType {
		return errors.NotSupportedf("%q auth-type", authType)
	}
	return nil
}
