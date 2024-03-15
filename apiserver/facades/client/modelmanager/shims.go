// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelmanager

import (
	"context"

	"github.com/juju/errors"
	"github.com/juju/names/v5"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/cloud"
	coremodel "github.com/juju/juju/core/model"
	"github.com/juju/juju/core/network"
	"github.com/juju/juju/domain/secretbackend"
	"github.com/juju/juju/environs/space"
)

type spaceStateShim struct {
	common.ModelManagerBackend
}

func (s spaceStateShim) AllSpaces() ([]network.SpaceInfo, error) {
	spaces, err := s.ModelManagerBackend.AllSpaces()
	if err != nil {
		return nil, errors.Trace(err)
	}

	results := make([]network.SpaceInfo, len(spaces))
	for i, space := range spaces {
		spaceInfo, err := space.NetworkSpace()
		if err != nil {
			return nil, errors.Trace(err)
		}
		results[i] = spaceInfo
	}
	return results, nil
}

func (s spaceStateShim) AddSpace(name string, providerId network.Id, subnetIds []string) (network.SpaceInfo, error) {
	result, err := s.ModelManagerBackend.AddSpace(name, providerId, subnetIds)
	if err != nil {
		return network.SpaceInfo{}, errors.Trace(err)
	}
	spaceInfo, err := result.NetworkSpace()
	if err != nil {
		return network.SpaceInfo{}, errors.Trace(err)
	}
	return spaceInfo, nil
}

func (s spaceStateShim) ConstraintsBySpaceName(name string) ([]space.Constraints, error) {
	constraints, err := s.ModelManagerBackend.ConstraintsBySpaceName(name)
	if err != nil {
		return nil, errors.Trace(err)
	}

	results := make([]space.Constraints, len(constraints))
	for i, constraint := range constraints {
		results[i] = constraint
	}
	return results, nil
}

func (s spaceStateShim) Remove(spaceID string) error {
	space, err := s.ModelManagerBackend.Space(spaceID)
	if err != nil {
		return errors.Trace(err)
	}
	return space.Remove()
}

type credentialStateShim struct {
	StateBackend
}

func (s credentialStateShim) CloudCredentialTag() (names.CloudCredentialTag, bool, error) {
	m, err := s.StateBackend.Model()
	if err != nil {
		return names.CloudCredentialTag{}, false, errors.Trace(err)
	}
	credTag, exists := m.CloudCredentialTag()
	return credTag, exists, nil
}

// SecretBackendService is an interface for interacting with secret backend service.
type SecretBackendService interface {
	BackendSummaryInfo(
		ctx context.Context,
		_ coremodel.UUID, _ secretbackend.ModelGetter, _ cloud.Cloud, _ cloud.Credential,
		reveal bool, _ secretbackend.SecretBackendFilter,
	) ([]*secretbackend.SecretBackendInfo, error)
}
