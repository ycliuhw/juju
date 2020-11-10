// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ecs

import (
	"github.com/juju/juju/caas"
)

// GetClusterMetadata implements ClusterMetadataChecker.
func (env *environ) GetClusterMetadata(storageClass string) (*caas.ClusterMetadata, error) {
	return nil, nil
}

// CheckDefaultWorkloadStorage implements ClusterMetadataChecker.
func (env *environ) CheckDefaultWorkloadStorage(cloudType string, storageProvisioner *caas.StorageProvisioner) error {
	return nil
}
