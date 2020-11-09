// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ecs

import (
	"github.com/juju/errors"
	"github.com/juju/jsonschema"
	"github.com/juju/loggo"

	"github.com/juju/juju/caas"
	"github.com/juju/juju/cloud"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/context"
)

var logger = loggo.GetLogger("juju.ecs.provider")

type environProvider struct {
	providerCredentials
}

// Version is part of the EnvironProvider interface.
func (environProvider) Version() int {
	return 0
}

// Open is specified in the EnvironProvider interface.
func (p environProvider) Open(args environs.OpenParams) (caas.Broker, error) {
	// new(clusterName string, cloud cloudspec.CloudSpec, envCfg *config.Config) (*environ, error)
	attr := args.Cloud.Credential.Attributes()
	if attr == nil {
		return nil, errors.NotValidf("empty credential %q", args.Cloud.Name)
	}
	clusterName := attr[CredAttrClusterName]
	if clusterName == "" {
		return nil, errors.NotValidf("empty cluster name %q", args.Cloud.Name)
	}
	return new(clusterName, args.Cloud, args.Config)
}

// CloudSchema returns the schema used to validate input for add-cloud.  Since
// this provider does not support custom clouds, this always returns nil.
func (p environProvider) CloudSchema() *jsonschema.Schema {
	return nil
}

// Ping tests the connection to the cloud, to verify the endpoint is valid.
func (p environProvider) Ping(ctx context.ProviderCallContext, endpoint string) error {
	return errors.NotImplementedf("Ping")
}

// PrepareConfig is specified in the EnvironProvider interface.
func (p environProvider) PrepareConfig(args environs.PrepareConfigParams) (*config.Config, error) {
	return nil, nil
}

// Validate is specified in the EnvironProvider interface.
func (environProvider) Validate(cfg, old *config.Config) (valid *config.Config, err error) {
	return nil, nil

}

// DetectRegions is specified in the environs.CloudRegionDetector interface.
func (p environProvider) DetectRegions() ([]cloud.Region, error) {
	return nil, errors.NotFoundf("regions")
}
