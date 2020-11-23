// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ecs

import (
	// jujuclock "github.com/juju/clock"
	"github.com/juju/errors"
	// "github.com/juju/jsonschema"
	// "github.com/juju/loggo"
	"github.com/juju/schema"
	// "github.com/juju/version"
	"gopkg.in/juju/environschema.v1"

	// "github.com/juju/juju/caas"
	"github.com/juju/juju/cloud"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
	// "github.com/juju/juju/environs/context"
)

const (
// // ClusterNameKey is the cloud config attribute used to specify
// // the cluster name for ecs cloud.
// ClusterNameKey = "cluster-name"
)

var configSchema = environschema.Fields{
	// ClusterNameKey: {
	// 	Description: "The name of the ECS cluster used to provision workload.",
	// 	Type:        environschema.Tstring,
	// 	Group:       environschema.AccountGroup,
	// },
}

var providerConfigFields = func() schema.Fields {
	fs, _, err := configSchema.ValidationSchema()
	if err != nil {
		panic(err)
	}
	return fs
}()

var providerConfigDefaults = schema.Defaults{
	// ClusterNameKey: "",
}

type brokerConfig struct {
	*config.Config
	attrs map[string]interface{}
}

// func (c *brokerConfig) clusterName() string {
// 	return c.attrs[ClusterNameKey].(string)
// }

func (p environProvider) Validate(cfg, old *config.Config) (*config.Config, error) {
	newCfg, err := validateConfig(cfg, old)
	if err != nil {
		return nil, errors.NewNotValid(err, "invalid ECS provider config")
	}
	return newCfg.Apply(newCfg.attrs)
}

func (p environProvider) newConfig(cfg *config.Config) (*brokerConfig, error) {
	valid, err := p.Validate(cfg, nil)
	if err != nil {
		return nil, err
	}
	return &brokerConfig{valid, valid.UnknownAttrs()}, nil
}

// Schema returns the configuration schema for an environment.
func (environProvider) Schema() environschema.Fields {
	fields, err := config.Schema(configSchema)
	if err != nil {
		panic(err)
	}
	return fields
}

// ConfigSchema returns extra config attributes specific
// to this provider only.
func (p environProvider) ConfigSchema() schema.Fields {
	return providerConfigFields
}

// ConfigDefaults returns the default values for the
// provider specific config attributes.
func (p environProvider) ConfigDefaults() schema.Defaults {
	return providerConfigDefaults
}

func validateConfig(cfg, old *config.Config) (*brokerConfig, error) {
	logger.Criticalf("validateConfig cfg -> %#v, old -> %#v", cfg, old)
	// Check for valid changes for the base config values.
	if err := config.Validate(cfg, old); err != nil {
		return nil, err
	}
	validated, err := cfg.ValidateUnknownAttrs(providerConfigFields, providerConfigDefaults)
	if err != nil {
		return nil, err
	}
	return &brokerConfig{Config: cfg, attrs: validated}, nil
}

// FinalizeCloud is part of the environs.CloudFinalizer interface.
func (p environProvider) FinalizeCloud(ctx environs.FinalizeCloudContext, cld cloud.Cloud) (cloud.Cloud, error) {
	return cld, nil
}
