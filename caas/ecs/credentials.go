// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ecs

import (
	"github.com/juju/juju/cloud"
	"github.com/juju/juju/environs"
)

const (
	CredAttrClusterName = "cluster-name"
)

type providerCredentials struct{}

// CredentialSchemas is part of the environs.ProviderCredentials interface.
func (providerCredentials) CredentialSchemas() map[cloud.AuthType]cloud.CredentialSchema {
	return nil
}

// DetectCredentials is part of the environs.ProviderCredentials interface.
func (e providerCredentials) DetectCredentials() (*cloud.CloudCredential, error) {
	return nil, nil
}

// FinalizeCredential is part of the environs.ProviderCredentials interface.
func (providerCredentials) FinalizeCredential(_ environs.FinalizeCredentialContext, args environs.FinalizeCredentialParams) (*cloud.Credential, error) {
	return &args.Credential, nil
}
