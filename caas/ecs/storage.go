// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ecs

import (
	"github.com/juju/errors"

	"github.com/juju/juju/caas"
	"github.com/juju/juju/caas/ecs/constants"
	jujustorage "github.com/juju/juju/storage"
)

const (

	// EBSVolumeType (default standard):
	//   "gp2" for General Purpose (SSD) volumes
	//   "io1" for Provisioned IOPS (SSD) volumes,
	//   "standard" for Magnetic volumes.
	EBSVolumeType = "volume-type"

	// Volume Aliases
	volumeAliasMagnetic = "magnetic" // standard
	volumeAliasSSD      = "ssd"      // gp2
)

// StorageProvider is defined on the jujustorage.ProviderRegistry interface.
func (env *environ) StorageProvider(t jujustorage.ProviderType) (jujustorage.Provider, error) {
	if t == constants.StorageProviderType {
		return &storageProvider{env}, nil
	}
	return nil, errors.NotFoundf("storage provider %q", t)
}

// EnsureStorageProvisioner creates a storage class with the specified config, or returns an existing one.
func (*environ) EnsureStorageProvisioner(cfg caas.StorageProvisioner) (*caas.StorageProvisioner, bool, error) {
	// REMOVE!!!
	return nil, false, nil
}

// StorageProviderTypes is defined on the jujustorage.ProviderRegistry interface.
func (*environ) StorageProviderTypes() ([]jujustorage.ProviderType, error) {
	return []jujustorage.ProviderType{constants.StorageProviderType}, nil
}

// ValidateStorageClass returns an error if the storage config is not valid.
func (*environ) ValidateStorageClass(config map[string]interface{}) error {
	// REMOVE!!!
	return nil
}

type storageProvider struct {
	client *environ
}

var _ jujustorage.Provider = (*storageProvider)(nil)

// ValidateConfig is defined on the jujustorage.Provider interface.
func (g *storageProvider) ValidateConfig(cfg *jujustorage.Config) error {
	// return errors.Trace(validateStorageAttributes(cfg.Attrs()))
	return nil
}

// Supports is defined on the jujustorage.Provider interface.
func (g *storageProvider) Supports(k jujustorage.StorageKind) bool {
	return k == jujustorage.StorageKindBlock ||
		k == jujustorage.StorageKindFilesystem // !!!
}

// Scope is defined on the jujustorage.Provider interface.
func (g *storageProvider) Scope() jujustorage.Scope {
	return jujustorage.ScopeEnviron
}

// Dynamic is defined on the jujustorage.Provider interface.
func (g *storageProvider) Dynamic() bool {
	return true
}

// Releasable is defined on the jujustorage.Provider interface.
func (g *storageProvider) Releasable() bool {
	return true
}

// DefaultPools is defined on the jujustorage.Provider interface.
func (g *storageProvider) DefaultPools() []*jujustorage.Config {
	ssdPool, _ := jujustorage.NewConfig("ecs-docker-volume-gb2", constants.StorageProviderType, map[string]interface{}{
		EBSVolumeType: volumeAliasSSD,
	})
	return []*jujustorage.Config{ssdPool}
}

// VolumeSource is defined on the jujustorage.Provider interface.
func (g *storageProvider) VolumeSource(cfg *jujustorage.Config) (jujustorage.VolumeSource, error) {
	// return &volumeSource{
	// 	client: g.client,
	// }, nil
	return nil, nil
}

// FilesystemSource is defined on the jujustorage.Provider interface.
func (g *storageProvider) FilesystemSource(providerConfig *jujustorage.Config) (jujustorage.FilesystemSource, error) {
	return nil, errors.NotSupportedf("filesystems")
}
