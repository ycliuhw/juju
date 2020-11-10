// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ecs

import (
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/juju/clock"

	"github.com/juju/juju/caas"
	"github.com/juju/juju/core/watcher"
)

type app struct {
	name           string
	clusterName    string
	modelUUID      string
	modelName      string
	deploymentType caas.DeploymentType
	client         ecsiface.ECSAPI
	clock          clock.Clock
}

func newApplication(
	name string,
	clusterName string,
	modelUUID string,
	modelName string,
	deploymentType caas.DeploymentType,
	client ecsiface.ECSAPI,
	clock clock.Clock,
) caas.Application {
	return &app{
		name:           name,
		clusterName:    clusterName,
		modelUUID:      modelUUID,
		modelName:      modelName,
		deploymentType: deploymentType,
		client:         client,
		clock:          clock,
	}
}

// Delete deletes the specified application.
func (a *app) Delete() error {
	return nil
}

// Ensure creates or updates an application pod with the given application
// name, agent path, and application config.
func (a *app) Ensure(config caas.ApplicationConfig) (err error) {
	return nil
}

// Exists indicates if the application for the specified
// application exists, and whether the application is terminating.
func (a *app) Exists() (caas.DeploymentState, error) {
	return caas.DeploymentState{}, nil
}

func (a *app) State() (caas.ApplicationState, error) {
	return caas.ApplicationState{}, nil
}

// Units of the application fetched from kubernetes by matching pod labels.
func (a *app) Units() ([]caas.Unit, error) {
	return nil, nil
}

// UpdatePorts updates port mappings on the specified service.
func (a *app) UpdatePorts(ports []caas.ServicePort, updateContainerPorts bool) error {
	return nil
}

// UpdateService updates the default service with specific service type and port mappings.
func (a *app) UpdateService(param caas.ServiceParam) error {
	return nil
}

// Watch returns a watcher which notifies when there
// are changes to the application of the specified application.
func (a *app) Watch() (watcher.NotifyWatcher, error) {
	return nil, nil
}

func (a *app) WatchReplicas() (watcher.NotifyWatcher, error) {
	return nil, nil
}

func (a *app) registerTaskDefinition() error {
	// a.client.RegisterTaskDefinition()
	return nil
}

func (a *app) ensureECSService() error {
	return nil
}
