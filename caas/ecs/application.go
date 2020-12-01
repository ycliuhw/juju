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
	jujustorage "github.com/juju/juju/storage"
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
	// TODO: prefix modelName to all resource names?
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

func (a *app) volumeName(storageName string) string {
	return fmt.Sprintf("%s-%s", a.name, storageName)
}

// getMountPathForFilesystem returns mount path.
func getMountPathForFilesystem(idx int, appName string, fs jujustorage.KubernetesFilesystemParams) string {
	if fs.Attachment != nil {
		return fs.Attachment.Path
	}
	return fmt.Sprintf(
		"%s/fs/%s/%s/%d",
		"/var/lib/juju",
		appName, fs.StorageName, idx,
	)
}

func (a *app) handleFileSystems(filesystems []jujustorage.KubernetesFilesystemParams) (vols []*ecs.Volume, mounts []*ecs.MountPoint) {
	for idx, fs := range filesystems {
		vol := &ecs.Volume{
			Name: aws.String(a.volumeName(fs.StorageName)),
			DockerVolumeConfiguration: &ecs.DockerVolumeConfiguration{
				Scope:         aws.String("shared"),
				Autoprovision: aws.Bool(true),
				Driver:        aws.String("rexray/ebs"), // TODO: fs.Attributes["Driver"] ?????
				Labels:        a.labels(),               // TODO: merge with fs.ResourceTags !!!
				DriverOpts: map[string]*string{
					"volumetype": aws.String("gp2"),                                // TODO!!!
					"size":       aws.String(strconv.FormatUint(fs.Size/1024, 10)), // unit of size here should be `Gi`
				},
			},
		}
		vols = append(vols, vol)

		readOnly := false
		if fs.Attachment != nil {
			readOnly = fs.Attachment.ReadOnly
		}
		mounts = append(mounts, &ecs.MountPoint{
			ContainerPath: aws.String(getMountPathForFilesystem(
				idx, a.name, fs,
			)),
			SourceVolume: vol.Name,
			ReadOnly:     aws.Bool(readOnly),
		})
	}
	return vols, mounts
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

	volumes, volumeMounts := a.handleFileSystems(config.Filesystems)

	input := &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String(a.name),
		TaskRoleArn: aws.String(""),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			// init container
			{
				Name:             aws.String("charm-init"),
				Image:            aws.String(config.AgentImagePath),
				WorkingDirectory: aws.String(jujuDataDir),
				Cpu:              aws.Int64(10),
				Memory:           aws.Int64(512),
				Essential:        aws.Bool(false),
				EntryPoint:       strPtrSlice("/opt/k8sagent"),
				Command: strPtrSlice(
					"init",
					"--data-dir",
					jujuDataDir,
					"--bin-dir",
					"/charm/bin",
				),
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
					// DO we need this in init container?
					// {
					// 	ContainerPath: aws.String("/charm/containers"),
					// 	SourceVolume:  aws.String("charm-data-containers"),
					// },
				},
			},
		},
		Volumes: append(volumes, []*ecs.Volume{
			// TODO: ensure no vol.Name conflict.
			{
				Name: aws.String("var-lib-juju"),
				DockerVolumeConfiguration: &ecs.DockerVolumeConfiguration{
					Scope:  aws.String("task"),
					Driver: aws.String("local"),
					Labels: a.labels(),
				},
			},
			{
				Name: aws.String("charm-data-bin"),
				DockerVolumeConfiguration: &ecs.DockerVolumeConfiguration{
					Scope:  aws.String("task"),
					Driver: aws.String("local"),
					Labels: a.labels(),
				},
			},
		}...),
	}
	// container agent.
	charmContainerDefinition := &ecs.ContainerDefinition{
		Name:             aws.String("charm"),
		Image:            aws.String(config.AgentImagePath),
		WorkingDirectory: aws.String(jujuDataDir),
		Cpu:              aws.Int64(10),
		Memory:           aws.Int64(512),
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
				ContainerPath: aws.String(jujuDataDir),
				SourceVolume:  aws.String("var-lib-juju"),
			},
			{
				ContainerPath: aws.String("/charm/bin"),
				SourceVolume:  aws.String("charm-data-bin"),
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
			MountPoints: append(volumeMounts,
				// TODO: ensure no vol.Name conflict.
				&ecs.MountPoint{
					ContainerPath: aws.String("/charm/bin"),
					SourceVolume:  aws.String("charm-data-bin"),
					ReadOnly:      aws.Bool(true),
				},
			),
		}
		volume := &ecs.Volume{
			Name: aws.String(fmt.Sprintf("charm-data-container-%s", v.Name)),
			DockerVolumeConfiguration: &ecs.DockerVolumeConfiguration{
				Scope:  aws.String("task"),
				Driver: aws.String("local"),
				Labels: a.labels(),
			},
		}
		input.Volumes = append(input.Volumes, volume)
		container.MountPoints = append(container.MountPoints, &ecs.MountPoint{
			ContainerPath: aws.String("/charm/container"),
			SourceVolume:  volume.Name,
		})
		input.ContainerDefinitions = append(input.ContainerDefinitions, container)
		charmContainerDefinition.MountPoints = append(charmContainerDefinition.MountPoints, &ecs.MountPoint{
			ContainerPath: aws.String(fmt.Sprintf("/charm/containers/%s", v.Name)),
			SourceVolume:  volume.Name,
		})
	}
	input.ContainerDefinitions = append(input.ContainerDefinitions, charmContainerDefinition)
	return input, nil
}

// Ensure creates or updates an application pod with the given application
// name, agent path, and application config.
func (a *app) Ensure(config caas.ApplicationConfig) (err error) {
	logger.Criticalf("app.Ensure config -> %s", pretty.Sprint(config))
	logger.Criticalf("app.Ensure a.labels() -> %s", pretty.Sprint(a.labels()))
	result, err := a.registerTaskDefinition(config)
	if err != nil {
		return errors.Trace(err)
	}
	taskDefinitionID := fmt.Sprintf(
		"%s:%s",
		aws.StringValue(result.TaskDefinition.Family),
		strconv.FormatInt(aws.Int64Value(result.TaskDefinition.Revision), 10),
	)
	return errors.Trace(a.ensureECSService(taskDefinitionID))
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

func (a *app) registerTaskDefinition(config caas.ApplicationConfig) (*ecs.RegisterTaskDefinitionOutput, error) {
	input, err := a.applicationTaskDefinition(config)
	if err != nil {
		return nil, errors.Trace(err)
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
		return nil, errors.Trace(err)
	}
	return result, nil
}

func (a *app) ensureECSService(taskDefinitionID string) (err error) {
	input := &ecs.CreateServiceInput{
		Cluster:        aws.String(a.clusterName),
		DesiredCount:   aws.Int64(1),
		ServiceName:    aws.String(a.name),
		TaskDefinition: aws.String(taskDefinitionID),
	}

	result, err := a.client.CreateService(input)
	logger.Criticalf("app.ensureECSService %q err -> %#v result -> %s", taskDefinitionID, err, pretty.Sprint(result))
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
