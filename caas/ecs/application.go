// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ecs

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/kr/pretty"

	"github.com/juju/juju/caas"
	"github.com/juju/juju/core/paths"
	"github.com/juju/juju/core/watcher"
	"github.com/juju/juju/core/watcher/watchertest"
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
	// TODO: prefix-modelName to all resource names?
	// Because ecs doesnot have namespace!!!
	// name = modelName + "-" + name
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

func (a *app) labels() map[string]*string {
	// TODO
	return map[string]*string{
		"App":       aws.String(a.name),
		"ModelName": aws.String(a.modelName),
		"ModelUUID": aws.String(a.modelUUID),
	}
}

// Delete deletes the specified application.
func (a *app) Delete() error {
	return nil
}

func strPtrSlice(in ...string) (out []*string) {
	for _, v := range in {
		out = append(out, aws.String(v))
	}
	return out
}

func (a *app) applicationTaskDefinition(config caas.ApplicationConfig) (*ecs.RegisterTaskDefinitionInput, error) {

	jujuDataDir, err := paths.DataDir("kubernetes") // !!!
	if err != nil {
		return nil, errors.Trace(err)
	}

	var containerNames []string
	var containers []caas.ContainerConfig
	for _, v := range config.Containers {
		containerNames = append(containerNames, v.Name)
		containers = append(containers, v)
	}
	sort.Strings(containerNames)
	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})

	input := &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String(a.name),
		TaskRoleArn: aws.String(""),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			// init container
			{
				Name:       aws.String("charm-init"),
				Image:      aws.String(config.AgentImagePath),
				Cpu:        aws.Int64(10),
				Memory:     aws.Int64(512),
				Essential:  aws.Bool(false),
				EntryPoint: strPtrSlice("/opt/k8sagent"),
				Command: strPtrSlice(
					"init",
					"--data-dir",
					jujuDataDir,
					"--bin-dir",
					"/charm/bin",
				),
				WorkingDirectory: aws.String(jujuDataDir),
				Environment: []*ecs.KeyValuePair{
					{
						Name:  aws.String("JUJU_CONTAINER_NAMES"),
						Value: aws.String(strings.Join(containerNames, ",")),
					},
					{
						Name:  aws.String("JUJU_K8S_POD_NAME"),
						Value: aws.String("cockroachdb-0"),
					},
					{
						Name:  aws.String("JUJU_K8S_POD_UUID"),
						Value: aws.String("c83b286e-8f45-4dbf-b2a6-0c393d93bd6a"),
					},
					// appSecret
					{
						Name:  aws.String("JUJU_K8S_APPLICATION"),
						Value: aws.String(a.name),
					},
					{
						Name:  aws.String("JUJU_K8S_MODEL"),
						Value: aws.String(a.modelUUID),
					},
					{
						Name:  aws.String("JUJU_K8S_APPLICATION_PASSWORD"),
						Value: aws.String(config.IntroductionSecret),
					},
					{
						Name:  aws.String("JUJU_K8S_CONTROLLER_ADDRESSES"),
						Value: aws.String(config.ControllerAddresses),
					},
					{
						Name:  aws.String("JUJU_K8S_CONTROLLER_CA_CERT"),
						Value: aws.String(config.ControllerCertBundle),
					},
				},
				MountPoints: []*ecs.MountPoint{
					{
						ContainerPath: aws.String("/var/lib/juju"),
						SourceVolume:  aws.String("var-lib-juju"),
					},
					{
						ContainerPath: aws.String("/charm/bin"),
						SourceVolume:  aws.String("charm-data-bin"),
					},
					{
						ContainerPath: aws.String("/charm/container"),
						SourceVolume:  aws.String("charm-data-container"),
					},
				},
			},
			// container agent.
			{
				Name:   aws.String("charm"),
				Image:  aws.String(config.AgentImagePath),
				Cpu:    aws.Int64(10),
				Memory: aws.Int64(512),
				DependsOn: []*ecs.ContainerDependency{
					{
						ContainerName: aws.String("charm-init"),
						Condition:     aws.String("COMPLETE"),
					},
				},
				Essential:  aws.Bool(true),
				EntryPoint: strPtrSlice("/charm/bin/k8sagent"),
				Command: strPtrSlice(
					"unit",
					"--data-dir", jujuDataDir,
					"--charm-modified-version", strconv.Itoa(config.CharmModifiedVersion),
					"--append-env",
					"PATH=$PATH:/charm/bin",
				),
				// TODO: Health check/prob
				Environment: []*ecs.KeyValuePair{
					{
						Name:  aws.String("JUJU_CONTAINER_NAMES"),
						Value: aws.String(strings.Join(containerNames, ",")),
					},
					{
						Name: aws.String(
							"HTTP_PROBE_PORT", // constants.EnvAgentHTTPProbePort
						),
						Value: aws.String(
							"3856", // constants.AgentHTTPProbePort
						),
					},
				},
				MountPoints: []*ecs.MountPoint{
					{
						ContainerPath: aws.String("/var/lib/juju"),
						SourceVolume:  aws.String("var-lib-juju"),
					},
					{
						ContainerPath: aws.String("/charm/bin"),
						SourceVolume:  aws.String("charm-data-bin"),
					},
					{
						ContainerPath: aws.String("/charm/container"),
						SourceVolume:  aws.String("charm-data-container"),
					},
				},
			},
		},
		Volumes: []*ecs.Volume{
			// TODO!!!
			{
				// Host: &ecs.HostVolumeProperties{
				// 	SourcePath: aws.String("/opt/charm-data/var/lib/juju"),
				// },
				Name: aws.String("var-lib-juju"),
				DockerVolumeConfiguration: &ecs.DockerVolumeConfiguration{
					Scope:  aws.String("task"),
					Driver: aws.String("local"),
					Labels: a.labels(),
					// Autoprovision: aws.Bool(true),
				},
			},
			{
				// Host: &ecs.HostVolumeProperties{
				// 	SourcePath: aws.String("/opt/charm-data/bin"),
				// },
				Name: aws.String("charm-data-bin"),
				DockerVolumeConfiguration: &ecs.DockerVolumeConfiguration{
					Scope:  aws.String("task"),
					Driver: aws.String("local"),
					Labels: a.labels(),
					// Autoprovision: aws.Bool(true),
				},
			},
			{
				// Host: &ecs.HostVolumeProperties{
				// 	SourcePath: aws.String("/opt/charm-data/containers/cockroachdb"),
				// },
				Name: aws.String("charm-data-container"),
				DockerVolumeConfiguration: &ecs.DockerVolumeConfiguration{
					Scope:  aws.String("task"),
					Driver: aws.String("local"),
					Labels: a.labels(),
					// Autoprovision: aws.Bool(true),
				},
			},
		},
	}

	for _, v := range containers {
		// TODO: https://aws.amazon.com/blogs/compute/amazon-ecs-and-docker-volume-drivers-amazon-ebs/
		// to use EBS volumes, it requires some docker storage plugin installed in the
		// container instance!!!
		// disable persistence storage or Juju have to manage those plugins????
		container := &ecs.ContainerDefinition{
			Name:  aws.String(v.Name),
			Image: aws.String(v.Image.RegistryPath),
			DependsOn: []*ecs.ContainerDependency{
				{
					ContainerName: aws.String("charm"),
					Condition:     aws.String("START"),
				},
			},
			Cpu:        aws.Int64(10),
			Memory:     aws.Int64(512),
			Essential:  aws.Bool(true),
			EntryPoint: strPtrSlice("/charm/bin/pebble"),
			Command: strPtrSlice(
				"listen",
				"--socket", "/charm/container/pebble.sock",
				"--append-env", "PATH=$PATH:/charm/bin",
			),
			// TODO: Health check/prob
			Environment: []*ecs.KeyValuePair{
				{
					Name:  aws.String("JUJU_CONTAINER_NAME"),
					Value: aws.String(v.Name),
				},
			},
			MountPoints: []*ecs.MountPoint{
				{
					ContainerPath: aws.String("/var/lib/juju"),
					SourceVolume:  aws.String("var-lib-juju"),
				},
				{
					ContainerPath: aws.String("/charm/bin"),
					SourceVolume:  aws.String("charm-data-bin"),
				},
				{
					ContainerPath: aws.String("/charm/container"),
					SourceVolume:  aws.String("charm-data-container"),
				},
			},
		}
		input.ContainerDefinitions = append(input.ContainerDefinitions, container)
	}
	return input, nil
}

// Ensure creates or updates an application pod with the given application
// name, agent path, and application config.
func (a *app) Ensure(config caas.ApplicationConfig) (err error) {
	logger.Criticalf("app.Ensure config -> %s", pretty.Sprint(config))
	logger.Criticalf("app.Ensure a.labels() -> %s", pretty.Sprint(a.labels()))

	if err := a.registerTaskDefinition(config); err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(a.ensureECSService())
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
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	return watchertest.NewMockNotifyWatcher(ch), nil
}

func (a *app) WatchReplicas() (watcher.NotifyWatcher, error) {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	return watchertest.NewMockNotifyWatcher(ch), nil
}

func (a *app) registerTaskDefinition(config caas.ApplicationConfig) error {
	input, err := a.applicationTaskDefinition(config)
	if err != nil {
		return errors.Trace(err)
	}

	result, err := a.client.RegisterTaskDefinition(input)
	logger.Criticalf("app.Ensure err -> %#v result -> %s", err, pretty.Sprint(result))
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return errors.Trace(err)
	}
	return nil
}

func (a *app) ensureECSService() (err error) {
	input := &ecs.CreateServiceInput{
		Cluster:        aws.String(a.clusterName),
		DesiredCount:   aws.Int64(1),
		ServiceName:    aws.String(a.name),
		TaskDefinition: aws.String(a.name),
	}

	result, err := a.client.CreateService(input)
	logger.Criticalf("app.ensureECSService err -> %#v result -> %s", err, pretty.Sprint(result))
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			case ecs.ErrCodeClusterNotFoundException:
				fmt.Println(ecs.ErrCodeClusterNotFoundException, aerr.Error())
			case ecs.ErrCodeUnsupportedFeatureException:
				fmt.Println(ecs.ErrCodeUnsupportedFeatureException, aerr.Error())
			case ecs.ErrCodePlatformUnknownException:
				fmt.Println(ecs.ErrCodePlatformUnknownException, aerr.Error())
			case ecs.ErrCodePlatformTaskDefinitionIncompatibilityException:
				fmt.Println(ecs.ErrCodePlatformTaskDefinitionIncompatibilityException, aerr.Error())
			case ecs.ErrCodeAccessDeniedException:
				fmt.Println(ecs.ErrCodeAccessDeniedException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}
	return errors.Trace(err)
}
