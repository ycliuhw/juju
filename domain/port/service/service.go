// Copyright 2024 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package service

import (
	"context"
	"fmt"

	"github.com/juju/errors"

	"github.com/juju/juju/core/network"
	"github.com/juju/juju/domain"
	"github.com/juju/juju/domain/port"
)

// WildcardEndpoint is a special endpoint that represents all endpoints.
const WildcardEndpoint = ""

// AtomicState describes the subset of methods on state that run within an atomic
// context.
type AtomicState interface {
	domain.AtomicStateBase

	// GetColocatedOpenedPorts returns all the open ports for all units co-located
	// with the given unit. Units are considered co-located if they share the same
	// net-node.
	GetColocatedOpenedPorts(ctx domain.AtomicContext, unitUUID string) ([]network.PortRange, error)

	// GetOpenedEndpointPorts returns the opened ports for a given endpoint of a
	// given unit.
	GetOpenedEndpointPorts(ctx domain.AtomicContext, unitUUID, endpoint string) ([]network.PortRange, error)

	// GetEndpoints returns all endpoints for a given unit.
	GetEndpoints(ctx domain.AtomicContext, unitUUID string) ([]string, error)

	// UpdateUnitPorts opens and closes ports for the endpoints of a given unit.
	// The opened and closed ports for the same endpoints must not conflict.
	UpdateUnitPorts(ctx domain.AtomicContext, unitUUID string, openPorts, closePorts network.GroupedPortRanges) error
}

// State describes the methods that a state implementation must provide to
// manage opened ports for units.
type State interface {
	AtomicState

	// GetUnitOpenedPorts returns the opened ports for a given unit uuid,
	// grouped by endpoint.
	GetUnitOpenedPorts(ctx context.Context, unitUUID string) (network.GroupedPortRanges, error)

	// GetMachineOpenedPorts returns the opened ports for all the units on the
	// machine. Opened ports are grouped first by unit and then by endpoint.
	GetMachineOpenedPorts(ctx context.Context, machineUUID string) (map[string]network.GroupedPortRanges, error)
}

// Service provides the API for managing the opened ports for units.
type Service struct {
	st State
}

// NewService returns a new Service providing an API to manage the opened ports
// for units.
func NewService(st State) *Service {
	return &Service{
		st: st,
	}
}

// GetUnitOpenedPorts returns the opened ports for a given unit uuid, grouped by endpoint.
func (s *Service) GetUnitOpenedPorts(ctx context.Context, unitUUID string) (network.GroupedPortRanges, error) {
	return s.st.GetUnitOpenedPorts(ctx, unitUUID)
}

// GetMachineOpenedPorts returns the opened ports for all the units on the machine.
// Opened ports are grouped first by unit and then by endpoint.
func (s *Service) GetMachineOpenedPorts(ctx context.Context, machineUUID string) (map[string]network.GroupedPortRanges, error) {
	return s.st.GetMachineOpenedPorts(ctx, machineUUID)
}

// UpdateUnitPorts opens and closes ports for the endpoints of a given unit.
//
// NOTE: There is a special wildcard endpoint "" that represents all endpoints.
// Any operations applied to the wildcard endpoint will logically applied to all
// endpoints.
//
// That is, if we open a port range on the wildcard endpoint, we will open it as
// usual but as a side effect we close that port range on all other endpoints.
//
// On the other hand, if we close a specific endpoint's port range that is open
// on the wildcard endpoint, we will close it on the wildcard endpoint and open
// it on all other endpoints except the targeted endpoint.
func (s *Service) UpdateUnitPorts(ctx context.Context, unitUUID string, openPorts, closePorts network.GroupedPortRanges) error {
	if len(openPorts.UniquePortRanges())+len(closePorts.UniquePortRanges()) == 0 {
		return nil
	}

	allInputPortRanges := append(openPorts.UniquePortRanges(), closePorts.UniquePortRanges()...)
	//  verify input port ranges do not conflict with each other.
	err := verifyNoPortRangeConflicts(allInputPortRanges, allInputPortRanges)
	if err != nil {
		return errors.Annotate(err, "cannot update unit ports with conflict(s)")
	}

	// construct a map of all port ranges being closed to the endpoint they're
	// being closed on. Except the wildcard endpoint.
	closePortsToEndpointMap := make(map[network.PortRange]string)
	for endpoint, endpointClosePorts := range closePorts {
		for _, portRange := range endpointClosePorts {
			closePortsToEndpointMap[portRange] = endpoint
		}
	}
	// Ensure endpoints closed on the wildcard endpoint are not in the map.
	for _, wildcardClosePorts := range closePorts[WildcardEndpoint] {
		delete(closePortsToEndpointMap, wildcardClosePorts)
	}

	err = s.st.RunAtomic(ctx, func(ctx domain.AtomicContext) error {
		// Verify input port ranges do no conflict with any port ranges co-located
		// with the unit.
		colocatedOpened, err := s.st.GetColocatedOpenedPorts(ctx, unitUUID)
		if err != nil {
			return errors.Annotate(err, "failed to get co-located opened ports")
		}
		err = verifyNoPortRangeConflicts(allInputPortRanges, colocatedOpened)
		if err != nil {
			return errors.Annotate(err, "cannot update unit ports with conflict(s) on co-located units")
		}

		wildcardOpen, _ := openPorts[WildcardEndpoint]
		wildcardClose, _ := closePorts[WildcardEndpoint]

		wildcardOpened, err := s.st.GetOpenedEndpointPorts(ctx, unitUUID, WildcardEndpoint)
		if err != nil {
			return errors.Annotate(err, "failed to get opened ports for wildcard endpoint")
		}
		wildcardOpenedSet := map[network.PortRange]bool{}
		for _, portRange := range wildcardOpened {
			wildcardOpenedSet[portRange] = true
		}

		// Remove openPorts ranges that are already open on the wildcard endpoint.
		for endpoint, endpointOpenPorts := range openPorts {
			if endpoint == WildcardEndpoint {
				continue
			}
			for i, portRange := range endpointOpenPorts {
				if _, ok := wildcardOpenedSet[portRange]; ok {
					openPorts[endpoint] = append(openPorts[endpoint][:i], openPorts[endpoint][i+1:]...)
				}
			}
		}

		// cache for endpoints. We may need to list the existing endpoints 0, 1,
		// or n times. Cache the result and only fill it when we need it, to avoid
		// unnecessary calls.
		var endpoints []string

		// If we're opening a port range on the wildcard endpoint, we need to
		// close it on all other endpoints.
		//
		// NOTE: This ensures that is a port range is open on the wildcard
		// endpoint, it is closed on all other endpoints.
		for _, openPortRange := range wildcardOpen {
			if endpoints == nil {
				endpoints, err = s.st.GetEndpoints(ctx, unitUUID)
				if err != nil {
					return errors.Annotate(err, "failed to get unit endpoints")
				}
			}

			for _, endpoint := range endpoints {
				if endpoint == WildcardEndpoint {
					continue
				}
				delete(openPorts, endpoint)
				closePorts[endpoint] = append(closePorts[endpoint], openPortRange)
			}
		}

		// Close port ranges closed on the wildcard endpoint on all other endpoints.
		for _, closePortRange := range wildcardClose {
			if endpoints == nil {
				endpoints, err = s.st.GetEndpoints(ctx, unitUUID)
				if err != nil {
					return errors.Annotate(err, "failed to get unit endpoints")
				}
			}

			for _, endpoint := range endpoints {
				if endpoint == WildcardEndpoint {
					continue
				}
				closePorts[endpoint] = append(closePorts[endpoint], closePortRange)
			}
		}

		// If we're closing a port range for a specific endpoint which is open
		// on the wildcard endpoint, we need to close it on the wildcard endpoint
		// and open it on all other endpoints except the targeted endpoint.
		for _, portRange := range wildcardOpened {
			if endpoint, ok := closePortsToEndpointMap[portRange]; ok {
				if endpoints == nil {
					endpoints, err = s.st.GetEndpoints(ctx, unitUUID)
					if err != nil {
						return errors.Annotate(err, "failed to get unit endpoints")
					}
				}

				// This port range, open on the wildcard endpoint, is being closed
				// on some endpoint. We need to close it on the wildcard, and open
				// it on all endpoints other than the wildcard & targeted endpoint.
				closePorts[WildcardEndpoint] = append(closePorts[WildcardEndpoint], portRange)

				for _, otherEndpoint := range endpoints {
					if otherEndpoint == WildcardEndpoint || otherEndpoint == endpoint {
						continue
					}
					openPorts[otherEndpoint] = append(openPorts[otherEndpoint], portRange)
				}

				// Remove the port range from openPorts for the targeted endpoint.
				for i, otherPortRange := range openPorts[endpoint] {
					if otherPortRange == portRange {
						openPorts[endpoint] = append(openPorts[endpoint][:i], openPorts[endpoint][i+1:]...)
						break
					}
				}
			}
		}

		return s.st.UpdateUnitPorts(ctx, unitUUID, openPorts, closePorts)
	})
	return errors.Annotate(err, "failed to update unit ports")
}

// verifyNoPortRangeConflicts verifies the provided port ranges do not conflict
// with each other.
//
// A conflict occurs when two (or more) port ranges across all endpoints overlap,
// but are not equal.
func verifyNoPortRangeConflicts(rangesA, rangesB []network.PortRange) error {
	var conflicts []string
	for _, portRange := range rangesA {
		for _, otherPortRange := range rangesB {
			if portRange.ConflictsWith(otherPortRange) && portRange != otherPortRange {
				conflicts = append(conflicts, fmt.Sprintf("[%s, %s]", portRange, otherPortRange))
			}
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	return errors.Annotatef(port.ErrPortRangeConflict, "conflicting port ranges: %s", conflicts)
}
