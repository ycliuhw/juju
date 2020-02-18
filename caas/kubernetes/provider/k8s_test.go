// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package provider_test

import (
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	jujuclock "github.com/juju/clock"
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/names.v3"
	"gopkg.in/juju/worker.v1/workertest"
	apps "k8s.io/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sstorage "k8s.io/api/storage/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/tools/cache"

	"github.com/juju/juju/caas"
	"github.com/juju/juju/caas/kubernetes/provider"
	k8sspecs "github.com/juju/juju/caas/kubernetes/provider/specs"
	"github.com/juju/juju/caas/specs"
	"github.com/juju/juju/core/application"
	"github.com/juju/juju/core/constraints"
	"github.com/juju/juju/core/devices"
	"github.com/juju/juju/core/status"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/context"
	envtesting "github.com/juju/juju/environs/testing"
	"github.com/juju/juju/storage"
	"github.com/juju/juju/testing"
)

type K8sSuite struct {
	testing.BaseSuite
}

var _ = gc.Suite(&K8sSuite{})

func float64Ptr(f float64) *float64 {
	return &f
}

func int32Ptr(i int32) *int32 {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func (s *K8sSuite) TestPrepareWorkloadSpecNoConfigConfig(c *gc.C) {

	sa := &specs.ServiceAccountSpec{}
	sa.AutomountServiceAccountToken = boolPtr(true)

	podSpec := specs.PodSpec{
		ServiceAccount: sa,
	}

	podSpec.ProviderPod = &k8sspecs.K8sPodSpec{
		KubernetesResources: &k8sspecs.KubernetesResources{
			Pod: &k8sspecs.PodSpec{
				RestartPolicy:                 core.RestartPolicyOnFailure,
				ActiveDeadlineSeconds:         int64Ptr(10),
				TerminationGracePeriodSeconds: int64Ptr(20),
				SecurityContext: &core.PodSecurityContext{
					RunAsNonRoot:       boolPtr(true),
					SupplementalGroups: []int64{1, 2},
				},
				ReadinessGates: []core.PodReadinessGate{
					{ConditionType: core.PodInitialized},
				},
				DNSPolicy:   core.DNSClusterFirst,
				HostNetwork: true,
			},
		},
	}
	podSpec.Containers = []specs.ContainerSpec{
		{
			Name:            "test",
			Ports:           []specs.ContainerPort{{ContainerPort: 80, Protocol: "TCP"}},
			Image:           "juju/image",
			ImagePullPolicy: specs.PullPolicy("Always"),
			ProviderContainer: &k8sspecs.K8sContainerSpec{
				ReadinessProbe: &core.Probe{
					InitialDelaySeconds: 10,
					Handler:             core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/ready"}},
				},
				LivenessProbe: &core.Probe{
					SuccessThreshold: 20,
					Handler:          core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/liveready"}},
				},
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot: boolPtr(true),
					Privileged:   boolPtr(true),
				},
			},
		}, {
			Name:  "test2",
			Ports: []specs.ContainerPort{{ContainerPort: 8080, Protocol: "TCP"}},
			Image: "juju/image2",
		},
	}

	spec, err := provider.PrepareWorkloadSpec("app-name", "app-name", &podSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(provider.PodSpec(spec), jc.DeepEquals, core.PodSpec{
		RestartPolicy:                 core.RestartPolicyOnFailure,
		ActiveDeadlineSeconds:         int64Ptr(10),
		TerminationGracePeriodSeconds: int64Ptr(20),
		SecurityContext: &core.PodSecurityContext{
			RunAsNonRoot:       boolPtr(true),
			SupplementalGroups: []int64{1, 2},
		},
		ReadinessGates: []core.PodReadinessGate{
			{ConditionType: core.PodInitialized},
		},
		DNSPolicy:                    core.DNSClusterFirst,
		HostNetwork:                  true,
		ServiceAccountName:           "app-name",
		AutomountServiceAccountToken: boolPtr(true),
		InitContainers:               initContainers(),
		Containers: []core.Container{
			{
				Name:            "test",
				Image:           "juju/image",
				Ports:           []core.ContainerPort{{ContainerPort: int32(80), Protocol: core.ProtocolTCP}},
				ImagePullPolicy: core.PullAlways,
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot: boolPtr(true),
					Privileged:   boolPtr(true),
				},
				ReadinessProbe: &core.Probe{
					InitialDelaySeconds: 10,
					Handler:             core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/ready"}},
				},
				LivenessProbe: &core.Probe{
					SuccessThreshold: 20,
					Handler:          core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/liveready"}},
				},
				VolumeMounts: dataVolumeMounts(),
			}, {
				Name:  "test2",
				Image: "juju/image2",
				Ports: []core.ContainerPort{{ContainerPort: int32(8080), Protocol: core.ProtocolTCP}},
				// Defaults since not specified.
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot:             boolPtr(false),
					ReadOnlyRootFilesystem:   boolPtr(false),
					AllowPrivilegeEscalation: boolPtr(true),
				},
				VolumeMounts: dataVolumeMounts(),
			},
		},
		Volumes: dataVolumes(),
	})
}

func (s *K8sSuite) TestPrepareWorkloadSpecWithEnvAndEnvFrom(c *gc.C) {

	sa := &specs.ServiceAccountSpec{}
	sa.AutomountServiceAccountToken = boolPtr(true)

	podSpec := specs.PodSpec{
		ServiceAccount: sa,
	}

	podSpec.ProviderPod = &k8sspecs.K8sPodSpec{
		KubernetesResources: &k8sspecs.KubernetesResources{
			Pod: &k8sspecs.PodSpec{
				RestartPolicy:                 core.RestartPolicyOnFailure,
				ActiveDeadlineSeconds:         int64Ptr(10),
				TerminationGracePeriodSeconds: int64Ptr(20),
				SecurityContext: &core.PodSecurityContext{
					RunAsNonRoot:       boolPtr(true),
					SupplementalGroups: []int64{1, 2},
				},
				ReadinessGates: []core.PodReadinessGate{
					{ConditionType: core.PodInitialized},
				},
				DNSPolicy: core.DNSClusterFirst,
			},
		},
	}

	envVarThing := core.EnvVar{
		Name: "thing",
		ValueFrom: &core.EnvVarSource{
			SecretKeyRef: &core.SecretKeySelector{Key: "bar"},
		},
	}
	envVarThing.ValueFrom.SecretKeyRef.Name = "foo"

	envVarThing1 := core.EnvVar{
		Name: "thing1",
		ValueFrom: &core.EnvVarSource{
			ConfigMapKeyRef: &core.ConfigMapKeySelector{Key: "bar"},
		},
	}
	envVarThing1.ValueFrom.ConfigMapKeyRef.Name = "foo"

	envFromSourceSecret1 := core.EnvFromSource{
		SecretRef: &core.SecretEnvSource{Optional: boolPtr(true)},
	}
	envFromSourceSecret1.SecretRef.Name = "secret1"

	envFromSourceSecret2 := core.EnvFromSource{
		SecretRef: &core.SecretEnvSource{},
	}
	envFromSourceSecret2.SecretRef.Name = "secret2"

	envFromSourceConfigmap1 := core.EnvFromSource{
		ConfigMapRef: &core.ConfigMapEnvSource{Optional: boolPtr(true)},
	}
	envFromSourceConfigmap1.ConfigMapRef.Name = "configmap1"

	envFromSourceConfigmap2 := core.EnvFromSource{
		ConfigMapRef: &core.ConfigMapEnvSource{},
	}
	envFromSourceConfigmap2.ConfigMapRef.Name = "configmap2"

	podSpec.Containers = []specs.ContainerSpec{
		{
			Name:            "test",
			Ports:           []specs.ContainerPort{{ContainerPort: 80, Protocol: "TCP"}},
			Image:           "juju/image",
			ImagePullPolicy: specs.PullPolicy("Always"),
			Config: map[string]interface{}{
				"restricted": "yes",
				"secret1": map[string]interface{}{
					"secret": map[string]interface{}{
						"optional": bool(true),
						"name":     "secret1",
					},
				},
				"secret2": map[string]interface{}{
					"secret": map[string]interface{}{
						"name": "secret2",
					},
				},
				"special": "p@ssword's",
				"switch":  bool(true),
				"MY_NODE_NAME": map[string]interface{}{
					"field": map[string]interface{}{
						"path": "spec.nodeName",
					},
				},
				"my-resource-limit": map[string]interface{}{
					"resource": map[string]interface{}{
						"container-name": "container1",
						"resource":       "requests.cpu",
						"divisor":        "1m",
					},
				},
				"attr": "foo=bar; name[\"fred\"]=\"blogs\";",
				"configmap1": map[string]interface{}{
					"config-map": map[string]interface{}{
						"name":     "configmap1",
						"optional": bool(true),
					},
				},
				"configmap2": map[string]interface{}{
					"config-map": map[string]interface{}{
						"name": "configmap2",
					},
				},
				"float": float64(111.11111111),
				"thing1": map[string]interface{}{
					"config-map": map[string]interface{}{
						"key":  "bar",
						"name": "foo",
					},
				},
				"brackets": "[\"hello\", \"world\"]",
				"foo":      "bar",
				"int":      float64(111),
				"thing": map[string]interface{}{
					"secret": map[string]interface{}{
						"key":  "bar",
						"name": "foo",
					},
				},
			},
			ProviderContainer: &k8sspecs.K8sContainerSpec{
				ReadinessProbe: &core.Probe{
					InitialDelaySeconds: 10,
					Handler:             core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/ready"}},
				},
				LivenessProbe: &core.Probe{
					SuccessThreshold: 20,
					Handler:          core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/liveready"}},
				},
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot: boolPtr(true),
					Privileged:   boolPtr(true),
				},
			},
		}, {
			Name:  "test2",
			Ports: []specs.ContainerPort{{ContainerPort: 8080, Protocol: "TCP"}},
			Image: "juju/image2",
		},
	}

	spec, err := provider.PrepareWorkloadSpec("app-name", "app-name", &podSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(provider.PodSpec(spec), jc.DeepEquals, core.PodSpec{
		RestartPolicy:                 core.RestartPolicyOnFailure,
		ActiveDeadlineSeconds:         int64Ptr(10),
		TerminationGracePeriodSeconds: int64Ptr(20),
		SecurityContext: &core.PodSecurityContext{
			RunAsNonRoot:       boolPtr(true),
			SupplementalGroups: []int64{1, 2},
		},
		ReadinessGates: []core.PodReadinessGate{
			{ConditionType: core.PodInitialized},
		},
		DNSPolicy:                    core.DNSClusterFirst,
		ServiceAccountName:           "app-name",
		AutomountServiceAccountToken: boolPtr(true),
		InitContainers:               initContainers(),
		Containers: []core.Container{
			{
				Name:            "test",
				Image:           "juju/image",
				Ports:           []core.ContainerPort{{ContainerPort: int32(80), Protocol: core.ProtocolTCP}},
				ImagePullPolicy: core.PullAlways,
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot: boolPtr(true),
					Privileged:   boolPtr(true),
				},
				ReadinessProbe: &core.Probe{
					InitialDelaySeconds: 10,
					Handler:             core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/ready"}},
				},
				LivenessProbe: &core.Probe{
					SuccessThreshold: 20,
					Handler:          core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/liveready"}},
				},
				VolumeMounts: dataVolumeMounts(),
				Env: []core.EnvVar{
					{Name: "MY_NODE_NAME", ValueFrom: &core.EnvVarSource{FieldRef: &core.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
					{Name: "attr", Value: `foo=bar; name["fred"]="blogs";`},
					{Name: "brackets", Value: `["hello", "world"]`},
					{Name: "float", Value: "111.11111111"},
					{Name: "foo", Value: "bar"},
					{Name: "int", Value: "111"},
					{
						Name: "my-resource-limit",
						ValueFrom: &core.EnvVarSource{
							ResourceFieldRef: &core.ResourceFieldSelector{
								ContainerName: "container1",
								Resource:      "requests.cpu",
								Divisor:       resource.MustParse("1m"),
							},
						},
					},
					{Name: "restricted", Value: "yes"},
					{Name: "special", Value: "p@ssword's"},
					{Name: "switch", Value: "true"},
					envVarThing,
					envVarThing1,
				},
				EnvFrom: []core.EnvFromSource{
					envFromSourceConfigmap1,
					envFromSourceConfigmap2,
					envFromSourceSecret1,
					envFromSourceSecret2,
				},
			}, {
				Name:  "test2",
				Image: "juju/image2",
				Ports: []core.ContainerPort{{ContainerPort: int32(8080), Protocol: core.ProtocolTCP}},
				// Defaults since not specified.
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot:             boolPtr(false),
					ReadOnlyRootFilesystem:   boolPtr(false),
					AllowPrivilegeEscalation: boolPtr(true),
				},
				VolumeMounts: dataVolumeMounts(),
			},
		},
		Volumes: dataVolumes(),
	})
}

func (s *K8sSuite) TestPrepareWorkloadSpecWithInitContainers(c *gc.C) {
	podSpec := specs.PodSpec{}
	podSpec.Containers = []specs.ContainerSpec{
		{
			Name:            "test",
			Ports:           []specs.ContainerPort{{ContainerPort: 80, Protocol: "TCP"}},
			Image:           "juju/image",
			ImagePullPolicy: specs.PullPolicy("Always"),
			ProviderContainer: &k8sspecs.K8sContainerSpec{
				ReadinessProbe: &core.Probe{
					InitialDelaySeconds: 10,
					Handler:             core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/ready"}},
				},
				LivenessProbe: &core.Probe{
					SuccessThreshold: 20,
					Handler:          core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/liveready"}},
				},
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot: boolPtr(true),
					Privileged:   boolPtr(true),
				},
			},
		}, {
			Name:  "test2",
			Ports: []specs.ContainerPort{{ContainerPort: 8080, Protocol: "TCP"}},
			Image: "juju/image2",
		},
		{
			Name:            "test-init",
			Init:            true,
			Ports:           []specs.ContainerPort{{ContainerPort: 90, Protocol: "TCP"}},
			Image:           "juju/image-init",
			ImagePullPolicy: specs.PullPolicy("Always"),
			WorkingDir:      "/path/to/here",
			Command:         []string{"sh", "ls"},
		},
	}

	spec, err := provider.PrepareWorkloadSpec("app-name", "app-name", &podSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(provider.PodSpec(spec), jc.DeepEquals, core.PodSpec{
		Containers: []core.Container{
			{
				Name:            "test",
				Image:           "juju/image",
				Ports:           []core.ContainerPort{{ContainerPort: int32(80), Protocol: core.ProtocolTCP}},
				ImagePullPolicy: core.PullAlways,
				ReadinessProbe: &core.Probe{
					InitialDelaySeconds: 10,
					Handler:             core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/ready"}},
				},
				LivenessProbe: &core.Probe{
					SuccessThreshold: 20,
					Handler:          core.Handler{HTTPGet: &core.HTTPGetAction{Path: "/liveready"}},
				},
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot: boolPtr(true),
					Privileged:   boolPtr(true),
				},
				VolumeMounts: dataVolumeMounts(),
			}, {
				Name:  "test2",
				Image: "juju/image2",
				Ports: []core.ContainerPort{{ContainerPort: int32(8080), Protocol: core.ProtocolTCP}},
				// Defaults since not specified.
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot:             boolPtr(false),
					ReadOnlyRootFilesystem:   boolPtr(false),
					AllowPrivilegeEscalation: boolPtr(true),
				},
				VolumeMounts: dataVolumeMounts(),
			},
		},
		InitContainers: append([]core.Container{
			{
				Name:            "test-init",
				Image:           "juju/image-init",
				Ports:           []core.ContainerPort{{ContainerPort: int32(90), Protocol: core.ProtocolTCP}},
				WorkingDir:      "/path/to/here",
				Command:         []string{"sh", "ls"},
				ImagePullPolicy: core.PullAlways,
				// Defaults since not specified.
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot:             boolPtr(false),
					ReadOnlyRootFilesystem:   boolPtr(false),
					AllowPrivilegeEscalation: boolPtr(true),
				},
			},
		}, initContainers()...),
		Volumes: dataVolumes(),
	})
}

func getBasicPodspec() *specs.PodSpec {
	pSpecs := &specs.PodSpec{}
	pSpecs.Containers = []specs.ContainerSpec{{
		Name:         "test",
		Ports:        []specs.ContainerPort{{ContainerPort: 80, Protocol: "TCP"}},
		ImageDetails: specs.ImageDetails{ImagePath: "juju/image", Username: "fred", Password: "secret"},
		Command:      []string{"sh", "-c"},
		Args:         []string{"doIt", "--debug"},
		WorkingDir:   "/path/to/here",
		Config: map[string]interface{}{
			"foo":        "bar",
			"restricted": "yes",
			"bar":        true,
			"switch":     true,
			"brackets":   `["hello", "world"]`,
		},
	}, {
		Name:  "test2",
		Ports: []specs.ContainerPort{{ContainerPort: 8080, Protocol: "TCP", Name: "fred"}},
		Image: "juju/image2",
	}}
	return pSpecs
}

var basicServiceArg = &core.Service{
	ObjectMeta: v1.ObjectMeta{
		Name:        "app-name",
		Labels:      map[string]string{"juju-app": "app-name"},
		Annotations: map[string]string{"juju.io/controller": testing.ControllerTag.Id()}},
	Spec: core.ServiceSpec{
		Selector: map[string]string{"juju-app": "app-name"},
		Type:     "nodeIP",
		Ports: []core.ServicePort{
			{Port: 80, TargetPort: intstr.FromInt(80), Protocol: "TCP"},
			{Port: 8080, Protocol: "TCP", Name: "fred"},
		},
		LoadBalancerIP: "10.0.0.1",
		ExternalName:   "ext-name",
	},
}

var basicHeadlessServiceArg = &core.Service{
	ObjectMeta: v1.ObjectMeta{
		Name:   "app-name-endpoints",
		Labels: map[string]string{"juju-app": "app-name"},
		Annotations: map[string]string{
			"juju.io/controller": testing.ControllerTag.Id(),
			"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
		},
	},
	Spec: core.ServiceSpec{
		Selector:                 map[string]string{"juju-app": "app-name"},
		Type:                     "ClusterIP",
		ClusterIP:                "None",
		PublishNotReadyAddresses: true,
	},
}

func (s *K8sBrokerSuite) getOCIImageSecret(c *gc.C, annotations map[string]string) *core.Secret {
	secretData, err := provider.CreateDockerConfigJSON(&getBasicPodspec().Containers[0].ImageDetails)
	c.Assert(err, jc.ErrorIsNil)
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["juju.io/controller"] = testing.ControllerTag.Id()

	return &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:        "app-name-test-secret",
			Namespace:   "test",
			Labels:      map[string]string{"juju-app": "app-name"},
			Annotations: annotations,
		},
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{".dockerconfigjson": secretData},
	}
}

func (s *K8sSuite) TestPrepareWorkloadSpecConfigPairs(c *gc.C) {
	spec, err := provider.PrepareWorkloadSpec("app-name", "app-name", getBasicPodspec(), "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(provider.PodSpec(spec), jc.DeepEquals, core.PodSpec{
		ImagePullSecrets: []core.LocalObjectReference{{Name: "app-name-test-secret"}},
		InitContainers:   initContainers(),
		Containers: []core.Container{
			{
				Name:       "test",
				Image:      "juju/image",
				Ports:      []core.ContainerPort{{ContainerPort: int32(80), Protocol: core.ProtocolTCP}},
				Command:    []string{"sh", "-c"},
				Args:       []string{"doIt", "--debug"},
				WorkingDir: "/path/to/here",
				Env: []core.EnvVar{
					{Name: "bar", Value: "true"},
					{Name: "brackets", Value: `["hello", "world"]`},
					{Name: "foo", Value: "bar"},
					{Name: "restricted", Value: "yes"},
					{Name: "switch", Value: "true"},
				},
				// Defaults since not specified.
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot:             boolPtr(false),
					ReadOnlyRootFilesystem:   boolPtr(false),
					AllowPrivilegeEscalation: boolPtr(true),
				},
				VolumeMounts: dataVolumeMounts(),
			}, {
				Name:  "test2",
				Image: "juju/image2",
				Ports: []core.ContainerPort{{ContainerPort: int32(8080), Protocol: core.ProtocolTCP, Name: "fred"}},
				// Defaults since not specified.
				SecurityContext: &core.SecurityContext{
					RunAsNonRoot:             boolPtr(false),
					ReadOnlyRootFilesystem:   boolPtr(false),
					AllowPrivilegeEscalation: boolPtr(true),
				},
				VolumeMounts: dataVolumeMounts(),
			},
		},
		Volumes: dataVolumes(),
	})
}

type K8sBrokerSuite struct {
	BaseSuite
}

var _ = gc.Suite(&K8sBrokerSuite{})

func (s *K8sBrokerSuite) TestAPIVersion(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	gomock.InOrder(
		s.mockDiscovery.EXPECT().ServerVersion().Return(&k8sversion.Info{
			Major: "1", Minor: "16",
		}, nil),
	)

	ver, err := s.broker.APIVersion()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(ver, gc.DeepEquals, "1.16.0")
}

func (s *K8sBrokerSuite) TestConfig(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	c.Assert(s.broker.Config(), jc.DeepEquals, s.cfg)
}

func (s *K8sBrokerSuite) TestSetConfig(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	err := s.broker.SetConfig(s.cfg)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestBootstrapNoOperatorStorage(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	ctx := envtesting.BootstrapContext(c)
	callCtx := &context.CloudCallContext{}
	bootstrapParams := environs.BootstrapParams{
		ControllerConfig:         testing.FakeControllerConfig(),
		BootstrapConstraints:     constraints.MustParse("mem=3.5G"),
		SupportedBootstrapSeries: testing.FakeSupportedJujuSeries,
	}

	_, err := s.broker.Bootstrap(ctx, callCtx, bootstrapParams)
	c.Assert(err, gc.NotNil)
	msg := strings.Replace(err.Error(), "\n", "", -1)
	c.Assert(msg, gc.Matches, "config without operator-storage value not valid.*")
}

func (s *K8sBrokerSuite) TestBootstrap(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	// Ensure the broker is configured with operator storage.
	s.setupOperatorStorageConfig(c)

	ctx := envtesting.BootstrapContext(c)
	callCtx := &context.CloudCallContext{}
	bootstrapParams := environs.BootstrapParams{
		ControllerConfig:         testing.FakeControllerConfig(),
		BootstrapConstraints:     constraints.MustParse("mem=3.5G"),
		SupportedBootstrapSeries: testing.FakeSupportedJujuSeries,
	}

	sc := &k8sstorage.StorageClass{
		ObjectMeta: v1.ObjectMeta{
			Name: "some-storage",
		},
	}
	gomock.InOrder(
		// Check the operator storage exists.
		s.mockStorageClass.EXPECT().Get("test-some-storage", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockStorageClass.EXPECT().Get("some-storage", v1.GetOptions{}).
			Return(sc, nil),
	)
	result, err := s.broker.Bootstrap(ctx, callCtx, bootstrapParams)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Arch, gc.Equals, "amd64")
	c.Assert(result.CaasBootstrapFinalizer, gc.NotNil)

	bootstrapParams.BootstrapSeries = "bionic"
	result, err = s.broker.Bootstrap(ctx, callCtx, bootstrapParams)
	c.Assert(err, jc.Satisfies, errors.IsNotSupported)
}

func (s *K8sBrokerSuite) setupOperatorStorageConfig(c *gc.C) {
	cfg := s.broker.Config()
	var err error
	cfg, err = cfg.Apply(map[string]interface{}{"operator-storage": "some-storage"})
	c.Assert(err, jc.ErrorIsNil)
	err = s.broker.SetConfig(cfg)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestPrepareForBootstrap(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	// Ensure the broker is configured with operator storage.
	s.setupOperatorStorageConfig(c)

	sc := &k8sstorage.StorageClass{
		ObjectMeta: v1.ObjectMeta{
			Name: "some-storage",
		},
	}

	gomock.InOrder(
		s.mockNamespaces.EXPECT().Get("controller-ctrl-1", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockNamespaces.EXPECT().List(v1.ListOptions{}).
			Return(&core.NamespaceList{Items: []core.Namespace{}}, nil),
		s.mockStorageClass.EXPECT().Get("controller-ctrl-1-some-storage", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockStorageClass.EXPECT().Get("some-storage", v1.GetOptions{}).
			Return(sc, nil),
	)
	ctx := envtesting.BootstrapContext(c)
	c.Assert(
		s.broker.PrepareForBootstrap(ctx, "ctrl-1"), jc.ErrorIsNil,
	)
	c.Assert(s.broker.GetCurrentNamespace(), jc.DeepEquals, "controller-ctrl-1")
}

func (s *K8sBrokerSuite) TestPrepareForBootstrapAlreadyExistNamespaceError(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	ns := &core.Namespace{ObjectMeta: v1.ObjectMeta{Name: "controller-ctrl-1"}}
	s.ensureJujuNamespaceAnnotations(true, ns)
	gomock.InOrder(
		s.mockNamespaces.EXPECT().Get("controller-ctrl-1", v1.GetOptions{}).
			Return(ns, nil),
	)
	ctx := envtesting.BootstrapContext(c)
	c.Assert(
		s.broker.PrepareForBootstrap(ctx, "ctrl-1"), jc.Satisfies, errors.IsAlreadyExists,
	)
}

func (s *K8sBrokerSuite) TestPrepareForBootstrapAlreadyExistControllerAnnotations(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	ns := &core.Namespace{ObjectMeta: v1.ObjectMeta{Name: "controller-ctrl-1"}}
	s.ensureJujuNamespaceAnnotations(true, ns)
	gomock.InOrder(
		s.mockNamespaces.EXPECT().Get("controller-ctrl-1", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockNamespaces.EXPECT().List(v1.ListOptions{}).
			Return(&core.NamespaceList{Items: []core.Namespace{*ns}}, nil),
	)
	ctx := envtesting.BootstrapContext(c)
	c.Assert(
		s.broker.PrepareForBootstrap(ctx, "ctrl-1"), jc.Satisfies, errors.IsAlreadyExists,
	)
}

func (s *K8sBrokerSuite) TestGetNamespace(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	ns := &core.Namespace{ObjectMeta: v1.ObjectMeta{Name: "test"}}
	s.ensureJujuNamespaceAnnotations(false, ns)
	gomock.InOrder(
		s.mockNamespaces.EXPECT().Get("test", v1.GetOptions{}).
			Return(ns, nil),
	)

	out, err := s.broker.GetNamespace("test")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(out, jc.DeepEquals, ns)
}

func (s *K8sBrokerSuite) TestGetNamespaceNotFound(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	gomock.InOrder(
		s.mockNamespaces.EXPECT().Get("unknown-namespace", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
	)

	out, err := s.broker.GetNamespace("unknown-namespace")
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
	c.Assert(out, gc.IsNil)
}

func (s *K8sBrokerSuite) TestNamespaces(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	ns1 := s.ensureJujuNamespaceAnnotations(false, &core.Namespace{ObjectMeta: v1.ObjectMeta{Name: "test"}})
	ns2 := s.ensureJujuNamespaceAnnotations(false, &core.Namespace{ObjectMeta: v1.ObjectMeta{Name: "test2"}})
	gomock.InOrder(
		s.mockNamespaces.EXPECT().List(v1.ListOptions{}).
			Return(&core.NamespaceList{Items: []core.Namespace{*ns1, *ns2}}, nil),
	)

	result, err := s.broker.Namespaces()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result, jc.SameContents, []string{"test", "test2"})
}

func (s *K8sBrokerSuite) assertDestroy(c *gc.C, isController bool, destroyFunc func() error) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	ns := &core.Namespace{}
	ns.Name = "test"
	s.ensureJujuNamespaceAnnotations(isController, ns)
	namespaceWatcher, namespaceFirer := newKubernetesTestWatcher()
	s.k8sWatcherFn = newK8sWatcherFunc(namespaceWatcher)

	gomock.InOrder(
		s.mockNamespaces.EXPECT().Get("test", v1.GetOptions{}).
			Return(ns, nil),

		s.mockNamespaces.EXPECT().Delete("test", s.deleteOptions(v1.DeletePropagationForeground, "")).
			Return(nil),

		s.mockStorageClass.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-model==test"},
		).
			Return(s.k8sNotFoundError()),

		// still terminating.
		s.mockNamespaces.EXPECT().Get("test", v1.GetOptions{}).
			DoAndReturn(func(_, _ interface{}) (*core.Namespace, error) {
				namespaceFirer()
				return ns, nil
			}),
		// terminated, not found returned
		s.mockNamespaces.EXPECT().Get("test", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
	)

	c.Assert(destroyFunc(), jc.ErrorIsNil)
	for _, watcher := range s.watchers {
		c.Assert(workertest.CheckKilled(c, watcher), jc.ErrorIsNil)
	}
}

func (s *K8sBrokerSuite) TestDestroyController(c *gc.C) {
	s.assertDestroy(c, true, func() error {
		return s.broker.DestroyController(context.NewCloudCallContext(), testing.ControllerTag.Id())
	})
}

func (s *K8sBrokerSuite) TestDestroy(c *gc.C) {
	s.assertDestroy(c, false, func() error { return s.broker.Destroy(context.NewCloudCallContext()) })
}

func (s *K8sBrokerSuite) TestGetCurrentNamespace(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()
	c.Assert(s.broker.GetCurrentNamespace(), jc.DeepEquals, s.getNamespace())
}

func (s *K8sBrokerSuite) TestCreate(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	ns := s.ensureJujuNamespaceAnnotations(false, &core.Namespace{ObjectMeta: v1.ObjectMeta{Name: "test"}})
	gomock.InOrder(
		s.mockNamespaces.EXPECT().Create(ns).
			Return(ns, nil),
	)

	err := s.broker.Create(
		&context.CloudCallContext{},
		environs.CreateParams{},
	)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestCreateAlreadyExists(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	ns := s.ensureJujuNamespaceAnnotations(false, &core.Namespace{ObjectMeta: v1.ObjectMeta{Name: "test"}})
	gomock.InOrder(
		s.mockNamespaces.EXPECT().Create(ns).
			Return(nil, s.k8sAlreadyExistsError()),
	)

	err := s.broker.Create(
		&context.CloudCallContext{},
		environs.CreateParams{},
	)
	c.Assert(err, jc.Satisfies, errors.IsAlreadyExists)
}

func unitStatefulSetArg(numUnits int32, scName string, podSpec core.PodSpec) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju-app-uuid":      "appuuid",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{"juju-app": "app-name"},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"juju.io/controller":                       testing.ControllerTag.Id(),
					},
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: []core.PersistentVolumeClaim{{
				ObjectMeta: v1.ObjectMeta{
					Name: "database-appuuid",
					Annotations: map[string]string{
						"foo":          "bar",
						"juju-storage": "database",
					}},
				Spec: core.PersistentVolumeClaimSpec{
					StorageClassName: &scName,
					AccessModes:      []core.PersistentVolumeAccessMode{core.ReadWriteOnce},
					Resources: core.ResourceRequirements{
						Requests: core.ResourceList{
							core.ResourceStorage: resource.MustParse("100Mi"),
						},
					},
				},
			}},
			PodManagementPolicy: apps.ParallelPodManagement,
			ServiceName:         "app-name-endpoints",
		},
	}
}

func (s *K8sBrokerSuite) TestDeleteServiceForApplication(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: v1.ObjectMeta{
			Name:      "tfjobs.kubeflow.org",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name", "juju-model": "test"},
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   "kubeflow.org",
			Version: "v1alpha2",
			Versions: []apiextensionsv1beta1.CustomResourceDefinitionVersion{
				{Name: "v1", Served: true, Storage: true},
				{Name: "v1alpha2", Served: true, Storage: false},
			},
			Scope: "Namespaced",
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "tfjobs",
				Kind:     "TFJob",
				Singular: "tfjob",
			},
			Validation: &apiextensionsv1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &apiextensionsv1beta1.JSONSchemaProps{
					Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
						"tfReplicaSpecs": {
							Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
								"Worker": {
									Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
										"replicas": {
											Type:    "integer",
											Minimum: float64Ptr(1),
										},
									},
								},
								"PS": {
									Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
										"replicas": {
											Type: "integer", Minimum: float64Ptr(1),
										},
									},
								},
								"Chief": {
									Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
										"replicas": {
											Type:    "integer",
											Minimum: float64Ptr(1),
											Maximum: float64Ptr(1),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Delete operations below return a not found to ensure it's treated as a no-op.
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-test", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),

		s.mockServices.EXPECT().Delete("test", s.deleteOptions(v1.DeletePropagationForeground, "")).
			Return(s.k8sNotFoundError()),
		s.mockStatefulSets.EXPECT().Delete("test", s.deleteOptions(v1.DeletePropagationForeground, "")).
			Return(s.k8sNotFoundError()),
		s.mockServices.EXPECT().Delete("test-endpoints", s.deleteOptions(v1.DeletePropagationForeground, "")).
			Return(s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Delete("test", s.deleteOptions(v1.DeletePropagationForeground, "")).
			Return(s.k8sNotFoundError()),

		// delete secrets.
		s.mockSecrets.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test"},
		).Return(nil),

		// delete configmaps.
		s.mockConfigMaps.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test"},
		).Return(nil),

		// delete RBAC resources.
		s.mockRoleBindings.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test"},
		).Return(nil),
		s.mockClusterRoleBindings.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test,juju-model==test"},
		).Return(nil),
		s.mockRoles.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test"},
		).Return(nil),
		s.mockClusterRoles.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test,juju-model==test"},
		).Return(nil),
		s.mockServiceAccounts.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test"},
		).Return(nil),

		// list cluster wide all custom resource definitions for deleting custom resources.
		s.mockCustomResourceDefinition.EXPECT().List(v1.ListOptions{}).
			Return(&apiextensionsv1beta1.CustomResourceDefinitionList{Items: []apiextensionsv1beta1.CustomResourceDefinition{*crd}}, nil),
		// delete all custom resources for crd "v1".
		s.mockDynamicClient.EXPECT().Resource(
			schema.GroupVersionResource{
				Group:    crd.Spec.Group,
				Version:  "v1",
				Resource: crd.Spec.Names.Plural,
			},
		).Return(s.mockNamespaceableResourceClient),
		s.mockResourceClient.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test"},
		).Return(nil),
		// delete all custom resources for crd "v1alpha2".
		s.mockDynamicClient.EXPECT().Resource(
			schema.GroupVersionResource{
				Group:    crd.Spec.Group,
				Version:  "v1alpha2",
				Resource: crd.Spec.Names.Plural,
			},
		).Return(s.mockNamespaceableResourceClient),
		s.mockResourceClient.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test"},
		).Return(nil),

		// delete all custom resource definitions.
		s.mockCustomResourceDefinition.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test,juju-model==test"},
		).Return(nil),

		// delete all mutating webhook configurations.
		s.mockMutatingWebhookConfiguration.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test,juju-model==test"},
		).Return(nil),

		// delete all validating webhook configurations.
		s.mockValidatingWebhookConfiguration.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test,juju-model==test"},
		).Return(nil),

		// delete all ingress resources.
		s.mockIngressInterface.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test"},
		).Return(nil),

		// delete all daemon set resources.
		s.mockDaemonSets.EXPECT().DeleteCollection(
			s.deleteOptions(v1.DeletePropagationForeground, ""),
			v1.ListOptions{LabelSelector: "juju-app==test"},
		).Return(nil),
	)

	err := s.broker.DeleteService("test")
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceNoUnits(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	two := int32(2)
	dc := &apps.Deployment{ObjectMeta: v1.ObjectMeta{Name: "juju-unit-storage"}, Spec: apps.DeploymentSpec{Replicas: &two}}
	zero := int32(0)
	emptyDc := dc
	emptyDc.Spec.Replicas = &zero
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(dc, nil),
		s.mockDeployments.EXPECT().Update(emptyDc).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{}
	err := s.broker.EnsureService("app-name", nil, params, 0, nil)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceNoStorage(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	numUnits := int32(2)
	basicPodSpec := getBasicPodspec()
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)

	deploymentArg := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels: map[string]string{
						"juju-app": "app-name",
					},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"fred":               "mary",
						"juju.io/controller": testing.ControllerTag.Id(),
					},
				},
				Spec: podSpec,
			},
		},
	}
	serviceArg := &core.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
				"a":                  "b",
			}},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"juju-app": "app-name"},
			Type:     "nodeIP",
			Ports: []core.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(80), Protocol: "TCP"},
				{Port: 8080, Protocol: "TCP", Name: "fred"},
			},
			LoadBalancerIP: "10.0.0.1",
			ExternalName:   "ext-name",
		},
	}

	ociImageSecret := s.getOCIImageSecret(c, map[string]string{"fred": "mary"})
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(serviceArg).
			Return(nil, nil),
		s.mockDeployments.EXPECT().Update(deploymentArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Create(deploymentArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec:           basicPodSpec,
		OperatorImagePath: "operator/image-path",
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
			"fred":                 "mary",
		},
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
		"kubernetes-service-annotations":     map[string]interface{}{"a": "b"},
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceWithConfigMapAndSecretsCreate(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	numUnits := int32(2)
	basicPodSpec := getBasicPodspec()
	basicPodSpec.ConfigMaps = map[string]specs.ConfigMap{
		"myData": {
			"foo":   "bar",
			"hello": "world",
		},
	}
	basicPodSpec.ProviderPod = &k8sspecs.K8sPodSpec{
		KubernetesResources: &k8sspecs.KubernetesResources{
			Secrets: []k8sspecs.Secret{
				{
					Name: "build-robot-secret",
					Type: core.SecretTypeOpaque,
					StringData: map[string]string{
						"config.yaml": `
apiUrl: "https://my.api.com/api/v1"
username: fred
password: shhhh`[1:],
					},
				},
				{
					Name: "another-build-robot-secret",
					Type: core.SecretTypeOpaque,
					Data: map[string]string{
						"username": "YWRtaW4=",
						"password": "MWYyZDFlMmU2N2Rm",
					},
				},
			},
		},
	}

	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)

	deploymentArg := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
			}},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels: map[string]string{
						"juju-app": "app-name",
					},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"fred":               "mary",
						"juju.io/controller": testing.ControllerTag.Id(),
					},
				},
				Spec: podSpec,
			},
		},
	}
	serviceArg := &core.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
				"a":                  "b",
			}},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"juju-app": "app-name"},
			Type:     "nodeIP",
			Ports: []core.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(80), Protocol: "TCP"},
				{Port: 8080, Protocol: "TCP", Name: "fred"},
			},
			LoadBalancerIP: "10.0.0.1",
			ExternalName:   "ext-name",
		},
	}

	cm := &core.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "myData",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
			},
		},
		Data: map[string]string{
			"foo":   "bar",
			"hello": "world",
		},
	}
	secrets1 := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "build-robot-secret",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
			},
		},
		Type: core.SecretTypeOpaque,
		StringData: map[string]string{
			"config.yaml": `
apiUrl: "https://my.api.com/api/v1"
username: fred
password: shhhh`[1:],
		},
	}

	secrets2Data, err := provider.ProcessSecretData(
		map[string]string{
			"username": "YWRtaW4=",
			"password": "MWYyZDFlMmU2N2Rm",
		},
	)
	c.Assert(err, jc.ErrorIsNil)
	secrets2 := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "another-build-robot-secret",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
			},
		},
		Type: core.SecretTypeOpaque,
		Data: secrets2Data,
	}

	ociImageSecret := s.getOCIImageSecret(c, map[string]string{"fred": "mary"})
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),

		// ensure configmaps.
		s.mockConfigMaps.EXPECT().Create(cm).
			Return(cm, nil),

		// ensure secrets.
		s.mockSecrets.EXPECT().Create(secrets1).
			Return(secrets1, nil),
		s.mockSecrets.EXPECT().Create(secrets2).
			Return(secrets2, nil),

		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(serviceArg).
			Return(nil, nil),
		s.mockDeployments.EXPECT().Update(deploymentArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Create(deploymentArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec:           basicPodSpec,
		OperatorImagePath: "operator/image-path",
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
			"fred":                 "mary",
		},
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
		"kubernetes-service-annotations":     map[string]interface{}{"a": "b"},
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestVersion(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	gomock.InOrder(
		s.mockDiscovery.EXPECT().ServerVersion().Return(&k8sversion.Info{
			Major: "1", Minor: "15+",
		}, nil),
	)

	ver, err := s.broker.Version()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(ver, gc.DeepEquals, &version.Number{Major: 1, Minor: 15})
}

func (s *K8sBrokerSuite) TestEnsureServiceWithConfigMapAndSecretsUpdate(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	numUnits := int32(2)
	basicPodSpec := getBasicPodspec()
	basicPodSpec.ConfigMaps = map[string]specs.ConfigMap{
		"myData": {
			"foo":   "bar",
			"hello": "world",
		},
	}
	basicPodSpec.ProviderPod = &k8sspecs.K8sPodSpec{
		KubernetesResources: &k8sspecs.KubernetesResources{
			Secrets: []k8sspecs.Secret{
				{
					Name: "build-robot-secret",
					Type: core.SecretTypeOpaque,
					StringData: map[string]string{
						"config.yaml": `
apiUrl: "https://my.api.com/api/v1"
username: fred
password: shhhh`[1:],
					},
				},
				{
					Name: "another-build-robot-secret",
					Type: core.SecretTypeOpaque,
					Data: map[string]string{
						"username": "YWRtaW4=",
						"password": "MWYyZDFlMmU2N2Rm",
					},
				},
			},
		},
	}

	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)

	deploymentArg := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
			}},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels: map[string]string{
						"juju-app": "app-name",
					},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"fred":               "mary",
						"juju.io/controller": testing.ControllerTag.Id(),
					},
				},
				Spec: podSpec,
			},
		},
	}
	serviceArg := &core.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
				"a":                  "b",
			}},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"juju-app": "app-name"},
			Type:     "nodeIP",
			Ports: []core.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(80), Protocol: "TCP"},
				{Port: 8080, Protocol: "TCP", Name: "fred"},
			},
			LoadBalancerIP: "10.0.0.1",
			ExternalName:   "ext-name",
		},
	}

	cm := &core.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "myData",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
			},
		},
		Data: map[string]string{
			"foo":   "bar",
			"hello": "world",
		},
	}
	secrets1 := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "build-robot-secret",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
			},
		},
		Type: core.SecretTypeOpaque,
		StringData: map[string]string{
			"config.yaml": `
apiUrl: "https://my.api.com/api/v1"
username: fred
password: shhhh`[1:],
		},
	}

	secrets2Data, err := provider.ProcessSecretData(
		map[string]string{
			"username": "YWRtaW4=",
			"password": "MWYyZDFlMmU2N2Rm",
		},
	)
	c.Assert(err, jc.ErrorIsNil)
	secrets2 := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "another-build-robot-secret",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
			},
		},
		Type: core.SecretTypeOpaque,
		Data: secrets2Data,
	}

	ociImageSecret := s.getOCIImageSecret(c, map[string]string{"fred": "mary"})
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),

		// ensure configmaps.
		s.mockConfigMaps.EXPECT().Create(cm).
			Return(nil, s.k8sAlreadyExistsError()),
		s.mockConfigMaps.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&core.ConfigMapList{Items: []core.ConfigMap{*cm}}, nil),
		s.mockConfigMaps.EXPECT().Update(cm).
			Return(cm, nil),

		// ensure secrets.
		s.mockSecrets.EXPECT().Create(secrets1).
			Return(nil, s.k8sAlreadyExistsError()),
		s.mockSecrets.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&core.SecretList{Items: []core.Secret{*secrets1}}, nil),
		s.mockSecrets.EXPECT().Update(secrets1).
			Return(secrets1, nil),
		s.mockSecrets.EXPECT().Create(secrets2).
			Return(nil, s.k8sAlreadyExistsError()),
		s.mockSecrets.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&core.SecretList{Items: []core.Secret{*secrets2}}, nil),
		s.mockSecrets.EXPECT().Update(secrets2).
			Return(secrets2, nil),

		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(serviceArg).
			Return(nil, nil),
		s.mockDeployments.EXPECT().Update(deploymentArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Create(deploymentArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: basicPodSpec,
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
			"fred":                 "mary",
		},
		OperatorImagePath: "operator/image-path",
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
		"kubernetes-service-annotations":     map[string]interface{}{"a": "b"},
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceNoStorageStateful(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	basicPodSpec := getBasicPodspec()
	basicPodSpec.Service = &specs.ServiceSpec{
		ScalePolicy: "serial",
	}
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)

	numUnits := int32(2)
	statefulSetArg := &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju-app-uuid":      "appuuid",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{"juju-app": "app-name"},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"juju.io/controller":                       testing.ControllerTag.Id(),
					},
				},
				Spec: podSpec,
			},
			PodManagementPolicy: apps.PodManagementPolicyType("OrderedReady"),
			ServiceName:         "app-name-endpoints",
		},
	}

	serviceArg := *basicServiceArg
	serviceArg.Spec.Type = core.ServiceTypeClusterIP
	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(&serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(&serviceArg).
			Return(nil, nil),
		s.mockServices.EXPECT().Get("app-name-endpoints", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicHeadlessServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicHeadlessServiceArg).
			Return(nil, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(nil, s.k8sNotFoundError()),
		s.mockStatefulSets.EXPECT().Create(statefulSetArg).
			Return(nil, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(statefulSetArg, nil),
		s.mockStatefulSets.EXPECT().Update(statefulSetArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: basicPodSpec,
		Deployment: caas.DeploymentParams{
			DeploymentType: caas.DeploymentStateful,
		},
		OperatorImagePath: "operator/image-path",
		ResourceTags:      map[string]string{"juju-controller-uuid": testing.ControllerTag.Id()},
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceCustomType(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	basicPodSpec := getBasicPodspec()
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)

	numUnits := int32(2)
	statefulSetArg := &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju-app-uuid":      "appuuid",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{"juju-app": "app-name"},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"juju.io/controller":                       testing.ControllerTag.Id(),
					},
				},
				Spec: podSpec,
			},
			PodManagementPolicy: apps.ParallelPodManagement,
			ServiceName:         "app-name-endpoints",
		},
	}

	serviceArg := *basicServiceArg
	serviceArg.Spec.Type = core.ServiceTypeExternalName
	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(&appsv1.StatefulSet{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"juju-app-uuid": "appuuid"}}}, nil),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(&serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(&serviceArg).
			Return(nil, nil),
		s.mockServices.EXPECT().Get("app-name-endpoints", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicHeadlessServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicHeadlessServiceArg).
			Return(nil, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(statefulSetArg, nil),
		s.mockStatefulSets.EXPECT().Create(statefulSetArg).
			Return(nil, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(statefulSetArg, nil),
		s.mockStatefulSets.EXPECT().Update(statefulSetArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: basicPodSpec,
		Deployment: caas.DeploymentParams{
			ServiceType: caas.ServiceExternal,
		},
		OperatorImagePath: "operator/image-path",
		ResourceTags:      map[string]string{"juju-controller-uuid": testing.ControllerTag.Id()},
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceServiceWithoutPortsNotValid(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	serviceArg := *basicServiceArg
	serviceArg.Spec.Type = core.ServiceTypeExternalName
	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(&appsv1.StatefulSet{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"juju-app-uuid": "appuuid"}}}, nil),
		s.mockSecrets.EXPECT().Delete("app-name-test-secret", s.deleteOptions(v1.DeletePropagationForeground, "")).
			Return(nil),
	)
	caasPodSpec := getBasicPodspec()
	for k, v := range caasPodSpec.Containers {
		v.Ports = []specs.ContainerPort{}
		caasPodSpec.Containers[k] = v
	}
	c.Assert(caasPodSpec.OmitServiceFrontend, jc.IsFalse)
	for _, v := range caasPodSpec.Containers {
		c.Check(len(v.Ports), jc.DeepEquals, 0)
	}
	params := &caas.ServiceParams{
		PodSpec: caasPodSpec,
		Deployment: caas.DeploymentParams{
			DeploymentType: caas.DeploymentStateful,
			ServiceType:    caas.ServiceExternal,
		},
		OperatorImagePath: "operator/image-path",
		ResourceTags:      map[string]string{"juju-controller-uuid": testing.ControllerTag.Id()},
	}
	err := s.broker.EnsureService(
		"app-name",
		func(_ string, _ status.Status, _ string, _ map[string]interface{}) error { return nil },
		params, 2,
		application.ConfigAttributes{
			"kubernetes-service-loadbalancer-ip": "10.0.0.1",
			"kubernetes-service-externalname":    "ext-name",
		},
	)
	c.Assert(err, gc.ErrorMatches, `ports are required for kubernetes service "app-name"`)
}

func (s *K8sBrokerSuite) TestEnsureServiceWithServiceAccountNewRoleCreate(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	podSpec := getBasicPodspec()
	podSpec.ServiceAccount = &specs.ServiceAccountSpec{}
	podSpec.ServiceAccount.AutomountServiceAccountToken = boolPtr(true)
	podSpec.ServiceAccount.Rules = []specs.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "watch", "list"},
		},
	}

	numUnits := int32(2)
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", podSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)

	deploymentArg := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels: map[string]string{
						"juju-app": "app-name",
					},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"fred":               "mary",
						"juju.io/controller": testing.ControllerTag.Id(),
					},
				},
				Spec: provider.PodSpec(workloadSpec),
			},
		},
	}
	serviceArg := &core.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"a":                  "b",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"juju-app": "app-name"},
			Type:     "nodeIP",
			Ports: []core.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(80), Protocol: "TCP"},
				{Port: 8080, Protocol: "TCP", Name: "fred"},
			},
			LoadBalancerIP: "10.0.0.1",
			ExternalName:   "ext-name",
		},
	}

	svcAccount := &core.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		AutomountServiceAccountToken: boolPtr(true),
	}
	role := &rbacv1.Role{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
	rb := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "app-name",
			Kind: "Role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "app-name",
				Namespace: "test",
			},
		},
	}

	secretArg := s.getOCIImageSecret(c, map[string]string{"fred": "mary"})
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServiceAccounts.EXPECT().Create(svcAccount).Return(svcAccount, nil),
		s.mockRoles.EXPECT().Create(role).Return(role, nil),
		s.mockRoleBindings.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&rbacv1.RoleBindingList{Items: []rbacv1.RoleBinding{}}, nil),
		s.mockRoleBindings.EXPECT().Create(rb).Return(rb, nil),
		s.mockSecrets.EXPECT().Create(secretArg).Return(secretArg, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(serviceArg).
			Return(nil, nil),
		s.mockDeployments.EXPECT().Update(deploymentArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Create(deploymentArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: podSpec,
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
			"fred":                 "mary",
		},
		OperatorImagePath: "operator/image-path",
	}
	err = s.broker.EnsureService("app-name", func(_ string, _ status.Status, _ string, _ map[string]interface{}) error { return nil }, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
		"kubernetes-service-annotations":     map[string]interface{}{"a": "b"},
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceWithServiceAccountNewRoleUpdate(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	podSpec := getBasicPodspec()
	podSpec.ServiceAccount = &specs.ServiceAccountSpec{}
	podSpec.ServiceAccount.AutomountServiceAccountToken = boolPtr(true)
	podSpec.ServiceAccount.Rules = []specs.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "watch", "list"},
		},
	}

	numUnits := int32(2)
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", podSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)

	deploymentArg := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels: map[string]string{
						"juju-app": "app-name",
					},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"fred":               "mary",
						"juju.io/controller": testing.ControllerTag.Id(),
					},
				},
				Spec: provider.PodSpec(workloadSpec),
			},
		},
	}
	serviceArg := &core.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"a":                  "b",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"juju-app": "app-name"},
			Type:     "nodeIP",
			Ports: []core.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(80), Protocol: "TCP"},
				{Port: 8080, Protocol: "TCP", Name: "fred"},
			},
			LoadBalancerIP: "10.0.0.1",
			ExternalName:   "ext-name",
		},
	}

	svcAccount := &core.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		AutomountServiceAccountToken: boolPtr(true),
	}
	role := &rbacv1.Role{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
	rb := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "app-name",
			Kind: "Role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "app-name",
				Namespace: "test",
			},
		},
	}
	rbUID := rb.GetUID()

	secretArg := s.getOCIImageSecret(c, map[string]string{"fred": "mary"})
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServiceAccounts.EXPECT().Create(svcAccount).Return(nil, s.k8sAlreadyExistsError()),
		s.mockServiceAccounts.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&core.ServiceAccountList{Items: []core.ServiceAccount{*svcAccount}}, nil),
		s.mockServiceAccounts.EXPECT().Update(svcAccount).Return(svcAccount, nil),
		s.mockRoles.EXPECT().Create(role).Return(nil, s.k8sAlreadyExistsError()),
		s.mockRoles.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&rbacv1.RoleList{Items: []rbacv1.Role{*role}}, nil),
		s.mockRoles.EXPECT().Update(role).Return(role, nil),
		s.mockRoleBindings.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&rbacv1.RoleBindingList{Items: []rbacv1.RoleBinding{*rb}}, nil),
		s.mockRoleBindings.EXPECT().Delete("app-name", s.deleteOptions(v1.DeletePropagationForeground, rbUID)).Return(nil),
		s.mockRoleBindings.EXPECT().Get("app-name", v1.GetOptions{}).Return(rb, nil),
		s.mockRoleBindings.EXPECT().Get("app-name", v1.GetOptions{}).Return(nil, s.k8sNotFoundError()),
		s.mockRoleBindings.EXPECT().Create(rb).Return(rb, nil),
		s.mockSecrets.EXPECT().Create(secretArg).Return(secretArg, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(serviceArg).
			Return(nil, nil),
		s.mockDeployments.EXPECT().Update(deploymentArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Create(deploymentArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: podSpec,
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
			"fred":                 "mary",
		},
		OperatorImagePath: "operator/image-path",
	}

	errChan := make(chan error)
	go func() {
		errChan <- s.broker.EnsureService("app-name", func(_ string, _ status.Status, _ string, _ map[string]interface{}) error { return nil }, params, 2, application.ConfigAttributes{
			"kubernetes-service-type":            "nodeIP",
			"kubernetes-service-loadbalancer-ip": "10.0.0.1",
			"kubernetes-service-externalname":    "ext-name",
			"kubernetes-service-annotations":     map[string]interface{}{"a": "b"},
		})
	}()
	err = s.clock.WaitAdvance(2*time.Second, testing.ShortWait, 1)
	c.Assert(err, jc.ErrorIsNil)

	select {
	case err := <-errChan:
		c.Assert(err, jc.ErrorIsNil)
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for EnsureService return")
	}
}

func (s *K8sBrokerSuite) TestEnsureServiceWithServiceAccountNewClusterRoleCreate(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	podSpec := getBasicPodspec()
	podSpec.ServiceAccount = &specs.ServiceAccountSpec{}
	podSpec.ServiceAccount.Global = true
	podSpec.ServiceAccount.AutomountServiceAccountToken = boolPtr(true)
	podSpec.ServiceAccount.Rules = []specs.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "watch", "list"},
		},
	}

	numUnits := int32(2)
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", podSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)

	deploymentArg := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels: map[string]string{
						"juju-app": "app-name",
					},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"fred":               "mary",
						"juju.io/controller": testing.ControllerTag.Id(),
					},
				},
				Spec: provider.PodSpec(workloadSpec),
			},
		},
	}
	serviceArg := &core.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"a":                  "b",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"juju-app": "app-name"},
			Type:     "nodeIP",
			Ports: []core.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(80), Protocol: "TCP"},
				{Port: 8080, Protocol: "TCP", Name: "fred"},
			},
			LoadBalancerIP: "10.0.0.1",
			ExternalName:   "ext-name",
		},
	}

	svcAccount := &core.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		AutomountServiceAccountToken: boolPtr(true),
	}
	cr := &rbacv1.ClusterRole{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name", "juju-model": "test"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name", "juju-model": "test"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "test-app-name",
			Kind: "ClusterRole",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "app-name",
				Namespace: "test",
			},
		},
	}

	secretArg := s.getOCIImageSecret(c, map[string]string{"fred": "mary"})
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServiceAccounts.EXPECT().Create(svcAccount).Return(svcAccount, nil),
		s.mockClusterRoles.EXPECT().Create(cr).Return(cr, nil),
		s.mockClusterRoleBindings.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name,juju-model==test"}).
			Return(&rbacv1.ClusterRoleBindingList{Items: []rbacv1.ClusterRoleBinding{}}, nil),
		s.mockClusterRoleBindings.EXPECT().Create(crb).Return(crb, nil),
		s.mockSecrets.EXPECT().Create(secretArg).Return(secretArg, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(serviceArg).
			Return(nil, nil),
		s.mockDeployments.EXPECT().Update(deploymentArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Create(deploymentArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: podSpec,
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
			"fred":                 "mary",
		},
		OperatorImagePath: "operator/image-path",
	}
	err = s.broker.EnsureService("app-name", func(_ string, _ status.Status, _ string, _ map[string]interface{}) error { return nil }, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
		"kubernetes-service-annotations":     map[string]interface{}{"a": "b"},
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceWithServiceAccountNewClusterRoleUpdate(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	podSpec := getBasicPodspec()
	podSpec.ServiceAccount = &specs.ServiceAccountSpec{}
	podSpec.ServiceAccount.Global = true
	podSpec.ServiceAccount.AutomountServiceAccountToken = boolPtr(true)
	podSpec.ServiceAccount.Rules = []specs.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "watch", "list"},
		},
	}

	numUnits := int32(2)
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", podSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)

	deploymentArg := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels: map[string]string{
						"juju-app": "app-name",
					},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"fred":               "mary",
						"juju.io/controller": testing.ControllerTag.Id(),
					},
				},
				Spec: provider.PodSpec(workloadSpec),
			},
		},
	}
	serviceArg := &core.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"a":                  "b",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"juju-app": "app-name"},
			Type:     "nodeIP",
			Ports: []core.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(80), Protocol: "TCP"},
				{Port: 8080, Protocol: "TCP", Name: "fred"},
			},
			LoadBalancerIP: "10.0.0.1",
			ExternalName:   "ext-name",
		},
	}

	svcAccount := &core.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		AutomountServiceAccountToken: boolPtr(true),
	}
	cr := &rbacv1.ClusterRole{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name", "juju-model": "test"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name", "juju-model": "test"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "test-app-name",
			Kind: "ClusterRole",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "app-name",
				Namespace: "test",
			},
		},
	}
	crbUID := crb.GetUID()

	secretArg := s.getOCIImageSecret(c, map[string]string{"fred": "mary"})
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServiceAccounts.EXPECT().Create(svcAccount).Return(nil, s.k8sAlreadyExistsError()),
		s.mockServiceAccounts.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&core.ServiceAccountList{Items: []core.ServiceAccount{*svcAccount}}, nil),
		s.mockServiceAccounts.EXPECT().Update(svcAccount).Return(svcAccount, nil),
		s.mockClusterRoles.EXPECT().Create(cr).Return(nil, s.k8sAlreadyExistsError()),
		s.mockClusterRoles.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name,juju-model==test"}).
			Return(&rbacv1.ClusterRoleList{Items: []rbacv1.ClusterRole{*cr}}, nil),
		s.mockClusterRoles.EXPECT().Update(cr).Return(cr, nil),
		s.mockClusterRoleBindings.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name,juju-model==test"}).
			Return(&rbacv1.ClusterRoleBindingList{Items: []rbacv1.ClusterRoleBinding{*crb}}, nil),
		s.mockClusterRoleBindings.EXPECT().Delete("test-app-name", s.deleteOptions(v1.DeletePropagationForeground, crbUID)).Return(nil),
		s.mockClusterRoleBindings.EXPECT().Get("test-app-name", v1.GetOptions{}).Return(crb, nil),
		s.mockClusterRoleBindings.EXPECT().Get("test-app-name", v1.GetOptions{}).Return(nil, s.k8sNotFoundError()),
		s.mockClusterRoleBindings.EXPECT().Create(crb).Return(crb, nil),
		s.mockSecrets.EXPECT().Create(secretArg).Return(secretArg, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(serviceArg).
			Return(nil, nil),
		s.mockDeployments.EXPECT().Update(deploymentArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Create(deploymentArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: podSpec,
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
			"fred":                 "mary",
		},
		OperatorImagePath: "operator/image-path",
	}

	errChan := make(chan error)
	go func() {
		errChan <- s.broker.EnsureService("app-name", func(_ string, _ status.Status, _ string, _ map[string]interface{}) error { return nil }, params, 2, application.ConfigAttributes{
			"kubernetes-service-type":            "nodeIP",
			"kubernetes-service-loadbalancer-ip": "10.0.0.1",
			"kubernetes-service-externalname":    "ext-name",
			"kubernetes-service-annotations":     map[string]interface{}{"a": "b"},
		})
	}()
	err = s.clock.WaitAdvance(2*time.Second, testing.ShortWait, 1)
	c.Assert(err, jc.ErrorIsNil)

	select {
	case err := <-errChan:
		c.Assert(err, jc.ErrorIsNil)
	case <-time.After(testing.LongWait):
		c.Fatalf("timed out waiting for EnsureService return")
	}
}

func (s *K8sBrokerSuite) TestEnsureServiceWithServiceAccountAndK8sServiceAccountNameSpaced(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	podSpec := getBasicPodspec()
	podSpec.ServiceAccount = &specs.ServiceAccountSpec{}
	podSpec.ServiceAccount.AutomountServiceAccountToken = boolPtr(true)
	podSpec.ServiceAccount.Rules = []specs.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "watch", "list"},
		},
	}

	k8sSASpec := k8sspecs.K8sServiceAccountSpec{
		Name: "k8sRBAC1",
	}
	k8sSASpec.AutomountServiceAccountToken = boolPtr(true)
	k8sSASpec.Rules = []specs.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "watch", "list"},
		},
	}
	podSpec.ProviderPod = &k8sspecs.K8sPodSpec{
		KubernetesResources: &k8sspecs.KubernetesResources{
			ServiceAccounts: []k8sspecs.K8sServiceAccountSpec{
				k8sSASpec,
			},
		},
	}

	numUnits := int32(2)
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", podSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)

	deploymentArg := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels: map[string]string{
						"juju-app": "app-name",
					},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"fred":               "mary",
						"juju.io/controller": testing.ControllerTag.Id(),
					},
				},
				Spec: provider.PodSpec(workloadSpec),
			},
		},
	}
	serviceArg := &core.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"a":                  "b",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"juju-app": "app-name"},
			Type:     "nodeIP",
			Ports: []core.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(80), Protocol: "TCP"},
				{Port: 8080, Protocol: "TCP", Name: "fred"},
			},
			LoadBalancerIP: "10.0.0.1",
			ExternalName:   "ext-name",
		},
	}

	svcAccount1 := &core.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		AutomountServiceAccountToken: boolPtr(true),
	}
	role1 := &rbacv1.Role{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
	rb1 := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "app-name",
			Kind: "Role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "app-name",
				Namespace: "test",
			},
		},
	}

	svcAccount2 := &core.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      "k8sRBAC1",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		AutomountServiceAccountToken: boolPtr(true),
	}
	role2 := &rbacv1.Role{
		ObjectMeta: v1.ObjectMeta{
			Name:      "k8sRBAC1",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
	rb2 := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      "k8sRBAC1",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "k8sRBAC1",
			Kind: "Role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "k8sRBAC1",
				Namespace: "test",
			},
		},
	}

	secretArg := s.getOCIImageSecret(c, map[string]string{"fred": "mary"})
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),

		s.mockServiceAccounts.EXPECT().Create(svcAccount1).Return(svcAccount1, nil),
		s.mockRoles.EXPECT().Create(role1).Return(role1, nil),
		s.mockRoleBindings.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&rbacv1.RoleBindingList{Items: []rbacv1.RoleBinding{}}, nil),
		s.mockRoleBindings.EXPECT().Create(rb1).Return(rb1, nil),

		s.mockServiceAccounts.EXPECT().Create(svcAccount2).Return(svcAccount2, nil),
		s.mockRoles.EXPECT().Create(role2).Return(role2, nil),
		s.mockRoleBindings.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&rbacv1.RoleBindingList{Items: []rbacv1.RoleBinding{}}, nil),
		s.mockRoleBindings.EXPECT().Create(rb2).Return(rb2, nil),

		s.mockSecrets.EXPECT().Create(secretArg).Return(secretArg, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(serviceArg).
			Return(nil, nil),
		s.mockDeployments.EXPECT().Update(deploymentArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Create(deploymentArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: podSpec,
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
			"fred":                 "mary",
		},
		OperatorImagePath: "operator/image-path",
	}
	err = s.broker.EnsureService("app-name", func(_ string, _ status.Status, _ string, _ map[string]interface{}) error { return nil }, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
		"kubernetes-service-annotations":     map[string]interface{}{"a": "b"},
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceWithServiceAccountAndK8sServiceAccountClusterScoped(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	podSpec := getBasicPodspec()
	podSpec.ServiceAccount = &specs.ServiceAccountSpec{}
	podSpec.ServiceAccount.AutomountServiceAccountToken = boolPtr(true)
	podSpec.ServiceAccount.Rules = []specs.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "watch", "list"},
		},
	}

	k8sSASpec := k8sspecs.K8sServiceAccountSpec{
		Name: "k8sRBAC1",
	}
	k8sSASpec.Global = true
	k8sSASpec.AutomountServiceAccountToken = boolPtr(true)
	k8sSASpec.Rules = []specs.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "watch", "list"},
		},
		{
			NonResourceURLs: []string{"/healthz", "/healthz/*"},
			Verbs:           []string{"get", "post"},
		},
		{
			APIGroups:     []string{"rbac.authorization.k8s.io"},
			Resources:     []string{"clusterroles"},
			Verbs:         []string{"bind"},
			ResourceNames: []string{"admin", "edit", "view"},
		},
	}
	podSpec.ProviderPod = &k8sspecs.K8sPodSpec{
		KubernetesResources: &k8sspecs.KubernetesResources{
			ServiceAccounts: []k8sspecs.K8sServiceAccountSpec{
				k8sSASpec,
			},
		},
	}

	numUnits := int32(2)
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", podSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)

	deploymentArg := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"juju.io/controller": testing.ControllerTag.Id(),
				"fred":               "mary",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels: map[string]string{
						"juju-app": "app-name",
					},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"juju.io/controller":                       testing.ControllerTag.Id(),
						"fred":                                     "mary",
					},
				},
				Spec: provider.PodSpec(workloadSpec),
			},
		},
	}
	serviceArg := &core.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:   "app-name",
			Labels: map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"a":                  "b",
				"juju.io/controller": testing.ControllerTag.Id(),
			}},
		Spec: core.ServiceSpec{
			Selector: map[string]string{"juju-app": "app-name"},
			Type:     "nodeIP",
			Ports: []core.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(80), Protocol: "TCP"},
				{Port: 8080, Protocol: "TCP", Name: "fred"},
			},
			LoadBalancerIP: "10.0.0.1",
			ExternalName:   "ext-name",
		},
	}

	svcAccount1 := &core.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		AutomountServiceAccountToken: boolPtr(true),
	}
	role1 := &rbacv1.Role{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
	rb1 := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-name",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "app-name",
			Kind: "Role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "app-name",
				Namespace: "test",
			},
		},
	}

	svcAccount2 := &core.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      "k8sRBAC1",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		AutomountServiceAccountToken: boolPtr(true),
	}
	clusterrole2 := &rbacv1.ClusterRole{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-k8sRBAC1",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name", "juju-model": "test"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				NonResourceURLs: []string{"/healthz", "/healthz/*"},
				Verbs:           []string{"get", "post"},
			},
			{
				APIGroups:     []string{"rbac.authorization.k8s.io"},
				Resources:     []string{"clusterroles"},
				Verbs:         []string{"bind"},
				ResourceNames: []string{"admin", "edit", "view"},
			},
		},
	}
	crb2 := &rbacv1.ClusterRoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-k8sRBAC1",
			Namespace: "test",
			Labels:    map[string]string{"juju-app": "app-name", "juju-model": "test"},
			Annotations: map[string]string{
				"fred":               "mary",
				"juju.io/controller": testing.ControllerTag.Id(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "test-k8sRBAC1",
			Kind: "ClusterRole",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "k8sRBAC1",
				Namespace: "test",
			},
		},
	}

	secretArg := s.getOCIImageSecret(c, map[string]string{"fred": "mary"})
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),

		s.mockServiceAccounts.EXPECT().Create(svcAccount1).Return(svcAccount1, nil),
		s.mockRoles.EXPECT().Create(role1).Return(role1, nil),
		s.mockRoleBindings.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name"}).
			Return(&rbacv1.RoleBindingList{Items: []rbacv1.RoleBinding{}}, nil),
		s.mockRoleBindings.EXPECT().Create(rb1).Return(rb1, nil),

		s.mockServiceAccounts.EXPECT().Create(svcAccount2).Return(svcAccount2, nil),
		s.mockClusterRoles.EXPECT().Create(clusterrole2).Return(clusterrole2, nil),
		s.mockClusterRoleBindings.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==app-name,juju-model==test"}).
			Return(&rbacv1.ClusterRoleBindingList{Items: []rbacv1.ClusterRoleBinding{}}, nil),
		s.mockClusterRoleBindings.EXPECT().Create(crb2).Return(crb2, nil),

		s.mockSecrets.EXPECT().Create(secretArg).Return(secretArg, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(serviceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(serviceArg).
			Return(nil, nil),
		s.mockDeployments.EXPECT().Update(deploymentArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Create(deploymentArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: podSpec,
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
			"fred":                 "mary",
		},
		OperatorImagePath: "operator/image-path",
	}
	err = s.broker.EnsureService("app-name", func(_ string, _ status.Status, _ string, _ map[string]interface{}) error { return nil }, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
		"kubernetes-service-annotations":     map[string]interface{}{"a": "b"},
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceWithStorage(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	basicPodSpec := getBasicPodspec()
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)
	podSpec.Containers[0].VolumeMounts = append(dataVolumeMounts(), core.VolumeMount{
		Name:      "database-appuuid",
		MountPath: "path/to/here",
	}, core.VolumeMount{
		Name:      "logs-1",
		MountPath: "path/to/there",
	})
	size, err := resource.ParseQuantity("200Mi")
	c.Assert(err, jc.ErrorIsNil)
	podSpec.Volumes = append(podSpec.Volumes, core.Volume{
		Name: "logs-1",
		VolumeSource: core.VolumeSource{EmptyDir: &core.EmptyDirVolumeSource{
			SizeLimit: &size,
			Medium:    "Memory",
		}},
	})
	statefulSetArg := unitStatefulSetArg(2, "workload-storage", podSpec)
	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(&appsv1.StatefulSet{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"juju-app-uuid": "appuuid"}}}, nil),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicServiceArg).
			Return(nil, nil),
		s.mockServices.EXPECT().Get("app-name-endpoints", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicHeadlessServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicHeadlessServiceArg).
			Return(nil, nil),
		s.mockStorageClass.EXPECT().Get("test-workload-storage", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockStorageClass.EXPECT().Get("workload-storage", v1.GetOptions{}).
			Return(&storagev1.StorageClass{ObjectMeta: v1.ObjectMeta{Name: "workload-storage"}}, nil),
		s.mockStatefulSets.EXPECT().Create(statefulSetArg).
			Return(nil, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(statefulSetArg, nil),
		s.mockStatefulSets.EXPECT().Update(statefulSetArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec:           basicPodSpec,
		OperatorImagePath: "operator/image-path",
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
		},
		Filesystems: []storage.KubernetesFilesystemParams{{
			StorageName: "database",
			Size:        100,
			Provider:    "kubernetes",
			Attributes:  map[string]interface{}{"storage-class": "workload-storage"},
			Attachment: &storage.KubernetesFilesystemAttachmentParams{
				Path: "path/to/here",
			},
			ResourceTags: map[string]string{"foo": "bar"},
		}, {
			StorageName: "logs",
			Size:        200,
			Provider:    "tmpfs",
			Attributes:  map[string]interface{}{"storage-medium": "Memory"},
			Attachment: &storage.KubernetesFilesystemAttachmentParams{
				Path: "path/to/there",
			},
		}},
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceForDeploymentWithDevices(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	numUnits := int32(2)
	basicPodSpec := getBasicPodspec()
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)
	podSpec.NodeSelector = map[string]string{"accelerator": "nvidia-tesla-p100"}
	for i := range podSpec.Containers {
		podSpec.Containers[i].Resources = core.ResourceRequirements{
			Limits: core.ResourceList{
				"nvidia.com/gpu": *resource.NewQuantity(3, resource.DecimalSI),
			},
			Requests: core.ResourceList{
				"nvidia.com/gpu": *resource.NewQuantity(3, resource.DecimalSI),
			},
		}
	}

	deploymentArg := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:        "app-name",
			Labels:      map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{"juju.io/controller": testing.ControllerTag.Id()}},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numUnits,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels:       map[string]string{"juju-app": "app-name"},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"juju.io/controller":                       testing.ControllerTag.Id(),
					},
				},
				Spec: podSpec,
			},
		},
	}
	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicServiceArg).
			Return(nil, nil),
		s.mockDeployments.EXPECT().Update(deploymentArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockDeployments.EXPECT().Create(deploymentArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec:           basicPodSpec,
		OperatorImagePath: "operator/image-path",
		Devices: []devices.KubernetesDeviceParams{
			{
				Type:       "nvidia.com/gpu",
				Count:      3,
				Attributes: map[string]string{"gpu": "nvidia-tesla-p100"},
			},
		},
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
		},
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceForDaemonSetWithDevicesAndConstraintsCreate(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	basicPodSpec := getBasicPodspec()
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)
	podSpec.NodeSelector = map[string]string{"accelerator": "nvidia-tesla-p100"}
	for i := range podSpec.Containers {
		podSpec.Containers[i].Resources = core.ResourceRequirements{
			Limits: core.ResourceList{
				"nvidia.com/gpu": *resource.NewQuantity(3, resource.DecimalSI),
			},
			Requests: core.ResourceList{
				"nvidia.com/gpu": *resource.NewQuantity(3, resource.DecimalSI),
			},
		}
	}
	podSpec.Affinity = &core.Affinity{
		NodeAffinity: &core.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &core.NodeSelector{
				NodeSelectorTerms: []core.NodeSelectorTerm{{
					MatchExpressions: []core.NodeSelectorRequirement{{
						Key:      "foo",
						Operator: core.NodeSelectorOpIn,
						Values:   []string{"a", "b", "c"},
					}, {
						Key:      "bar",
						Operator: core.NodeSelectorOpNotIn,
						Values:   []string{"d", "e", "f"},
					}, {
						Key:      "foo",
						Operator: core.NodeSelectorOpNotIn,
						Values:   []string{"g", "h"},
					}},
				}},
			},
		},
	}

	daemonSetArg := &appsv1.DaemonSet{
		ObjectMeta: v1.ObjectMeta{
			Name:        "app-name",
			Labels:      map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{"juju.io/controller": testing.ControllerTag.Id()}},
		Spec: appsv1.DaemonSetSpec{
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels:       map[string]string{"juju-app": "app-name"},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"juju.io/controller":                       testing.ControllerTag.Id(),
					},
				},
				Spec: podSpec,
			},
		},
	}

	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicServiceArg).
			Return(nil, nil),
		s.mockDaemonSets.EXPECT().Create(daemonSetArg).
			Return(daemonSetArg, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: basicPodSpec,
		Deployment: caas.DeploymentParams{
			DeploymentType: caas.DeploymentDaemon,
		},
		OperatorImagePath: "operator/image-path",
		Devices: []devices.KubernetesDeviceParams{
			{
				Type:       "nvidia.com/gpu",
				Count:      3,
				Attributes: map[string]string{"gpu": "nvidia-tesla-p100"},
			},
		},
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
		},
		Constraints: constraints.MustParse(`tags=foo=a|b|c,^bar=d|e|f,^foo=g|h`),
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceForDaemonSetWithDevicesAndConstraintsUpdate(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	basicPodSpec := getBasicPodspec()
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)
	podSpec.NodeSelector = map[string]string{"accelerator": "nvidia-tesla-p100"}
	for i := range podSpec.Containers {
		podSpec.Containers[i].Resources = core.ResourceRequirements{
			Limits: core.ResourceList{
				"nvidia.com/gpu": *resource.NewQuantity(3, resource.DecimalSI),
			},
			Requests: core.ResourceList{
				"nvidia.com/gpu": *resource.NewQuantity(3, resource.DecimalSI),
			},
		}
	}
	podSpec.Affinity = &core.Affinity{
		NodeAffinity: &core.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &core.NodeSelector{
				NodeSelectorTerms: []core.NodeSelectorTerm{{
					MatchExpressions: []core.NodeSelectorRequirement{{
						Key:      "foo",
						Operator: core.NodeSelectorOpIn,
						Values:   []string{"a", "b", "c"},
					}, {
						Key:      "bar",
						Operator: core.NodeSelectorOpNotIn,
						Values:   []string{"d", "e", "f"},
					}, {
						Key:      "foo",
						Operator: core.NodeSelectorOpNotIn,
						Values:   []string{"g", "h"},
					}},
				}},
			},
		},
	}

	daemonSetArg := &appsv1.DaemonSet{
		ObjectMeta: v1.ObjectMeta{
			Name:        "app-name",
			Labels:      map[string]string{"juju-app": "app-name"},
			Annotations: map[string]string{"juju.io/controller": testing.ControllerTag.Id()}},
		Spec: appsv1.DaemonSetSpec{
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"juju-app": "app-name"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "app-name-",
					Labels:       map[string]string{"juju-app": "app-name"},
					Annotations: map[string]string{
						"apparmor.security.beta.kubernetes.io/pod": "runtime/default",
						"seccomp.security.beta.kubernetes.io/pod":  "docker/default",
						"juju.io/controller":                       testing.ControllerTag.Id(),
					},
				},
				Spec: podSpec,
			},
		},
	}

	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicServiceArg).
			Return(nil, nil),
		s.mockDaemonSets.EXPECT().Create(daemonSetArg).
			Return(nil, s.k8sAlreadyExistsError()),
		s.mockDaemonSets.EXPECT().List(v1.ListOptions{
			LabelSelector:        "juju-app==app-name",
			IncludeUninitialized: true,
		}).Return(&appsv1.DaemonSetList{Items: []appsv1.DaemonSet{*daemonSetArg}}, nil),
		s.mockDaemonSets.EXPECT().Update(daemonSetArg).
			Return(daemonSetArg, nil),
	)

	params := &caas.ServiceParams{
		PodSpec: basicPodSpec,
		Deployment: caas.DeploymentParams{
			DeploymentType: caas.DeploymentDaemon,
		},
		OperatorImagePath: "operator/image-path",
		Devices: []devices.KubernetesDeviceParams{
			{
				Type:       "nvidia.com/gpu",
				Count:      3,
				Attributes: map[string]string{"gpu": "nvidia-tesla-p100"},
			},
		},
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
		},
		Constraints: constraints.MustParse(`tags=foo=a|b|c,^bar=d|e|f,^foo=g|h`),
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceForStatefulSetWithDevices(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	basicPodSpec := getBasicPodspec()
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)
	podSpec.Containers[0].VolumeMounts = append(dataVolumeMounts(), core.VolumeMount{
		Name:      "database-appuuid",
		MountPath: "path/to/here",
	})
	podSpec.NodeSelector = map[string]string{"accelerator": "nvidia-tesla-p100"}
	for i := range podSpec.Containers {
		podSpec.Containers[i].Resources = core.ResourceRequirements{
			Limits: core.ResourceList{
				"nvidia.com/gpu": *resource.NewQuantity(3, resource.DecimalSI),
			},
			Requests: core.ResourceList{
				"nvidia.com/gpu": *resource.NewQuantity(3, resource.DecimalSI),
			},
		}
	}
	statefulSetArg := unitStatefulSetArg(2, "workload-storage", podSpec)
	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(&appsv1.StatefulSet{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"juju-app-uuid": "appuuid"}}}, nil),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicServiceArg).
			Return(nil, nil),
		s.mockServices.EXPECT().Get("app-name-endpoints", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicHeadlessServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicHeadlessServiceArg).
			Return(nil, nil),
		s.mockStorageClass.EXPECT().Get("test-workload-storage", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockStorageClass.EXPECT().Get("workload-storage", v1.GetOptions{}).
			Return(&storagev1.StorageClass{ObjectMeta: v1.ObjectMeta{Name: "workload-storage"}}, nil),
		s.mockStatefulSets.EXPECT().Create(statefulSetArg).
			Return(statefulSetArg, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(statefulSetArg, nil),
		s.mockStatefulSets.EXPECT().Update(statefulSetArg).
			Return(statefulSetArg, nil),
	)

	params := &caas.ServiceParams{
		PodSpec:           basicPodSpec,
		OperatorImagePath: "operator/image-path",
		Filesystems: []storage.KubernetesFilesystemParams{{
			StorageName: "database",
			Size:        100,
			Provider:    "kubernetes",
			Attachment: &storage.KubernetesFilesystemAttachmentParams{
				Path: "path/to/here",
			},
			Attributes:   map[string]interface{}{"storage-class": "workload-storage"},
			ResourceTags: map[string]string{"foo": "bar"},
		}},
		Devices: []devices.KubernetesDeviceParams{
			{
				Type:       "nvidia.com/gpu",
				Count:      3,
				Attributes: map[string]string{"gpu": "nvidia-tesla-p100"},
			},
		},
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
		},
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceWithConstraints(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	basicPodSpec := getBasicPodspec()
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)
	podSpec.Containers[0].VolumeMounts = append(dataVolumeMounts(), core.VolumeMount{
		Name:      "database-appuuid",
		MountPath: "path/to/here",
	})
	for i := range podSpec.Containers {
		podSpec.Containers[i].Resources = core.ResourceRequirements{
			Limits: core.ResourceList{
				"memory": resource.MustParse("64Mi"),
				"cpu":    resource.MustParse("500m"),
			},
		}
	}
	statefulSetArg := unitStatefulSetArg(2, "workload-storage", podSpec)
	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(&appsv1.StatefulSet{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"juju-app-uuid": "appuuid"}}}, nil),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicServiceArg).
			Return(nil, nil),
		s.mockServices.EXPECT().Get("app-name-endpoints", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicHeadlessServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicHeadlessServiceArg).
			Return(nil, nil),
		s.mockStorageClass.EXPECT().Get("test-workload-storage", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockStorageClass.EXPECT().Get("workload-storage", v1.GetOptions{}).
			Return(&storagev1.StorageClass{ObjectMeta: v1.ObjectMeta{Name: "workload-storage"}}, nil),
		s.mockStatefulSets.EXPECT().Create(statefulSetArg).
			Return(nil, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(statefulSetArg, nil),
		s.mockStatefulSets.EXPECT().Update(statefulSetArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec:           basicPodSpec,
		OperatorImagePath: "operator/image-path",
		Filesystems: []storage.KubernetesFilesystemParams{{
			StorageName: "database",
			Size:        100,
			Provider:    "kubernetes",
			Attachment: &storage.KubernetesFilesystemAttachmentParams{
				Path: "path/to/here",
			},
			Attributes:   map[string]interface{}{"storage-class": "workload-storage"},
			ResourceTags: map[string]string{"foo": "bar"},
		}},
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
		},
		Constraints: constraints.MustParse("mem=64 cpu-power=500"),
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceWithNodeAffinity(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	basicPodSpec := getBasicPodspec()
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)
	podSpec.Containers[0].VolumeMounts = append(dataVolumeMounts(), core.VolumeMount{
		Name:      "database-appuuid",
		MountPath: "path/to/here",
	})
	podSpec.Affinity = &core.Affinity{
		NodeAffinity: &core.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &core.NodeSelector{
				NodeSelectorTerms: []core.NodeSelectorTerm{{
					MatchExpressions: []core.NodeSelectorRequirement{{
						Key:      "foo",
						Operator: core.NodeSelectorOpIn,
						Values:   []string{"a", "b", "c"},
					}, {
						Key:      "bar",
						Operator: core.NodeSelectorOpNotIn,
						Values:   []string{"d", "e", "f"},
					}, {
						Key:      "foo",
						Operator: core.NodeSelectorOpNotIn,
						Values:   []string{"g", "h"},
					}},
				}},
			},
		},
	}
	statefulSetArg := unitStatefulSetArg(2, "workload-storage", podSpec)
	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(&appsv1.StatefulSet{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"juju-app-uuid": "appuuid"}}}, nil),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicServiceArg).
			Return(nil, nil),
		s.mockServices.EXPECT().Get("app-name-endpoints", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicHeadlessServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicHeadlessServiceArg).
			Return(nil, nil),
		s.mockStorageClass.EXPECT().Get("test-workload-storage", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockStorageClass.EXPECT().Get("workload-storage", v1.GetOptions{}).
			Return(&storagev1.StorageClass{ObjectMeta: v1.ObjectMeta{Name: "workload-storage"}}, nil),
		s.mockStatefulSets.EXPECT().Create(statefulSetArg).
			Return(nil, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(statefulSetArg, nil),
		s.mockStatefulSets.EXPECT().Update(statefulSetArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec:           basicPodSpec,
		OperatorImagePath: "operator/image-path",
		Filesystems: []storage.KubernetesFilesystemParams{{
			StorageName: "database",
			Size:        100,
			Provider:    "kubernetes",
			Attachment: &storage.KubernetesFilesystemAttachmentParams{
				Path: "path/to/here",
			},
			Attributes:   map[string]interface{}{"storage-class": "workload-storage"},
			ResourceTags: map[string]string{"foo": "bar"},
		}},
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
		},
		Constraints: constraints.MustParse(`tags=foo=a|b|c,^bar=d|e|f,^foo=g|h`),
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestEnsureServiceWithZones(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	basicPodSpec := getBasicPodspec()
	workloadSpec, err := provider.PrepareWorkloadSpec("app-name", "app-name", basicPodSpec, "operator/image-path")
	c.Assert(err, jc.ErrorIsNil)
	podSpec := provider.PodSpec(workloadSpec)
	podSpec.Containers[0].VolumeMounts = append(dataVolumeMounts(), core.VolumeMount{
		Name:      "database-appuuid",
		MountPath: "path/to/here",
	})
	podSpec.Affinity = &core.Affinity{
		NodeAffinity: &core.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &core.NodeSelector{
				NodeSelectorTerms: []core.NodeSelectorTerm{{
					MatchExpressions: []core.NodeSelectorRequirement{{
						Key:      "failure-domain.beta.kubernetes.io/zone",
						Operator: core.NodeSelectorOpIn,
						Values:   []string{"a", "b", "c"},
					}},
				}},
			},
		},
	}
	statefulSetArg := unitStatefulSetArg(2, "workload-storage", podSpec)
	ociImageSecret := s.getOCIImageSecret(c, nil)
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockSecrets.EXPECT().Create(ociImageSecret).
			Return(ociImageSecret, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(&appsv1.StatefulSet{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"juju-app-uuid": "appuuid"}}}, nil),
		s.mockServices.EXPECT().Get("app-name", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicServiceArg).
			Return(nil, nil),
		s.mockServices.EXPECT().Get("app-name-endpoints", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Update(basicHeadlessServiceArg).
			Return(nil, s.k8sNotFoundError()),
		s.mockServices.EXPECT().Create(basicHeadlessServiceArg).
			Return(nil, nil),
		s.mockStorageClass.EXPECT().Get("test-workload-storage", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockStorageClass.EXPECT().Get("workload-storage", v1.GetOptions{}).
			Return(&storagev1.StorageClass{ObjectMeta: v1.ObjectMeta{Name: "workload-storage"}}, nil),
		s.mockStatefulSets.EXPECT().Create(statefulSetArg).
			Return(nil, nil),
		s.mockStatefulSets.EXPECT().Get("app-name", v1.GetOptions{IncludeUninitialized: true}).
			Return(statefulSetArg, nil),
		s.mockStatefulSets.EXPECT().Update(statefulSetArg).
			Return(nil, nil),
	)

	params := &caas.ServiceParams{
		PodSpec:           basicPodSpec,
		OperatorImagePath: "operator/image-path",
		Filesystems: []storage.KubernetesFilesystemParams{{
			StorageName: "database",
			Size:        100,
			Provider:    "kubernetes",
			Attachment: &storage.KubernetesFilesystemAttachmentParams{
				Path: "path/to/here",
			},
			Attributes:   map[string]interface{}{"storage-class": "workload-storage"},
			ResourceTags: map[string]string{"foo": "bar"},
		}},
		ResourceTags: map[string]string{
			"juju-controller-uuid": testing.ControllerTag.Id(),
		},
		Constraints: constraints.MustParse(`zones=a,b,c`),
	}
	err = s.broker.EnsureService("app-name", nil, params, 2, application.ConfigAttributes{
		"kubernetes-service-type":            "nodeIP",
		"kubernetes-service-loadbalancer-ip": "10.0.0.1",
		"kubernetes-service-externalname":    "ext-name",
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestWatchService(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	s.k8sWatcherFn = provider.NewK8sWatcherFunc(func(_ cache.SharedIndexInformer, _ string, _ jujuclock.Clock) (provider.KubernetesNotifyWatcher, error) {
		w, _ := newKubernetesTestWatcher()
		return w, nil
	})

	w, err := s.broker.WatchService("test")
	c.Assert(err, jc.ErrorIsNil)

	select {
	case _, ok := <-w.Changes():
		c.Assert(ok, jc.IsTrue)
	case <-time.After(testing.LongWait):
		c.Fatal("timed out waiting for event")
	}
}

func (s *K8sBrokerSuite) TestAnnotateUnit(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	pod := &core.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: "pod-name",
		},
	}

	updatePod := &core.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:        "pod-name",
			Annotations: map[string]string{"juju.io/unit": "appname/0"},
		},
	}

	gomock.InOrder(
		s.mockPods.EXPECT().Get("pod-name", v1.GetOptions{}).Return(pod, nil),
		s.mockPods.EXPECT().Update(updatePod).Return(updatePod, nil),
	)

	err := s.broker.AnnotateUnit("appname", "pod-name", names.NewUnitTag("appname/0"))
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestAnnotateUnitByUID(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	podList := &core.PodList{
		Items: []core.Pod{{ObjectMeta: v1.ObjectMeta{
			Name: "pod-name",
			UID:  types.UID("uuid"),
		}}},
	}

	updatePod := &core.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:        "pod-name",
			UID:         types.UID("uuid"),
			Annotations: map[string]string{"juju.io/unit": "appname/0"},
		},
	}

	gomock.InOrder(
		s.mockPods.EXPECT().Get("uuid", v1.GetOptions{}).Return(nil, s.k8sNotFoundError()),
		s.mockPods.EXPECT().List(v1.ListOptions{LabelSelector: "juju-app==appname"}).Return(podList, nil),
		s.mockPods.EXPECT().Update(updatePod).Return(updatePod, nil),
	)

	err := s.broker.AnnotateUnit("appname", "uuid", names.NewUnitTag("appname/0"))
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestWatchContainerStart(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	podWatcher, podFirer := newKubernetesTestStringsWatcher()
	var filter provider.K8sStringsWatcherFilterFunc
	s.k8sStringsWatcherFn = provider.NewK8sStringsWatcherFunc(
		func(_ cache.SharedIndexInformer,
			_ string,
			_ jujuclock.Clock,
			_ []string,
			ff provider.K8sStringsWatcherFilterFunc) (provider.KubernetesStringsWatcher, error) {
			filter = ff
			return podWatcher, nil
		},
	)

	podList := &core.PodList{
		Items: []core.Pod{{
			ObjectMeta: v1.ObjectMeta{
				Name: "test-0",
				OwnerReferences: []v1.OwnerReference{
					{Kind: "StatefulSet"},
				},
				Annotations: map[string]string{
					"juju.io/unit": "test-0",
				},
			},
			Status: core.PodStatus{
				InitContainerStatuses: []core.ContainerStatus{
					{Name: "juju-pod-init", State: core.ContainerState{Waiting: &core.ContainerStateWaiting{}}},
				},
				Phase: core.PodPending,
			},
		}},
	}

	gomock.InOrder(
		s.mockPods.EXPECT().List(
			listOptionsLabelSelectorMatcher("juju-app==test"),
		).DoAndReturn(func(...interface{}) (*core.PodList, error) {
			return podList, nil
		}),
	)

	w, err := s.broker.WatchContainerStart("test", caas.InitContainerName)
	c.Assert(err, jc.ErrorIsNil)

	select {
	case v, ok := <-w.Changes():
		c.Assert(ok, jc.IsTrue)
		c.Assert(v, gc.HasLen, 0)
	case <-time.After(testing.LongWait):
		c.Fatal("timed out waiting for event")
	}

	pod := &core.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: "test-0",
			OwnerReferences: []v1.OwnerReference{
				{Kind: "StatefulSet"},
			},
			Annotations: map[string]string{
				"juju.io/unit": "test-0",
			},
		},
		Status: core.PodStatus{
			InitContainerStatuses: []core.ContainerStatus{
				{Name: "juju-pod-init", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
			},
			Phase: core.PodPending,
		},
	}

	evt, ok := filter(provider.WatchEventUpdate, pod)
	c.Assert(ok, jc.IsTrue)
	podFirer([]string{evt})

	select {
	case v, ok := <-w.Changes():
		c.Assert(ok, jc.IsTrue)
		c.Assert(v, gc.DeepEquals, []string{"test-0"})
	case <-time.After(testing.LongWait):
		c.Fatal("timed out waiting for event")
	}
}

func (s *K8sBrokerSuite) TestWatchContainerStartDefault(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	podWatcher, podFirer := newKubernetesTestStringsWatcher()
	var filter provider.K8sStringsWatcherFilterFunc
	s.k8sStringsWatcherFn = provider.NewK8sStringsWatcherFunc(
		func(_ cache.SharedIndexInformer,
			_ string,
			_ jujuclock.Clock,
			_ []string,
			ff provider.K8sStringsWatcherFilterFunc) (provider.KubernetesStringsWatcher, error) {
			filter = ff
			return podWatcher, nil
		},
	)

	podList := &core.PodList{
		Items: []core.Pod{{
			ObjectMeta: v1.ObjectMeta{
				Name: "test-0",
				OwnerReferences: []v1.OwnerReference{
					{Kind: "StatefulSet"},
				},
				Annotations: map[string]string{
					"juju.io/unit": "test-0",
				},
			},
			Status: core.PodStatus{
				ContainerStatuses: []core.ContainerStatus{
					{Name: "first-container", State: core.ContainerState{Waiting: &core.ContainerStateWaiting{}}},
					{Name: "second-container", State: core.ContainerState{Waiting: &core.ContainerStateWaiting{}}},
				},
				Phase: core.PodPending,
			},
		}},
	}

	gomock.InOrder(
		s.mockPods.EXPECT().List(
			listOptionsLabelSelectorMatcher("juju-app==test"),
		).Return(podList, nil),
	)

	w, err := s.broker.WatchContainerStart("test", "")
	c.Assert(err, jc.ErrorIsNil)

	// Send an event to one of the watchers; multi-watcher should fire.
	pod := &core.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: "test-0",
			OwnerReferences: []v1.OwnerReference{
				{Kind: "StatefulSet"},
			},
			Annotations: map[string]string{
				"juju.io/unit": "test-0",
			},
		},
		Status: core.PodStatus{
			ContainerStatuses: []core.ContainerStatus{
				{Name: "first-container", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
				{Name: "second-container", State: core.ContainerState{Waiting: &core.ContainerStateWaiting{}}},
			},
			Phase: core.PodPending,
		},
	}

	select {
	case v, ok := <-w.Changes():
		c.Assert(ok, jc.IsTrue)
		c.Assert(v, gc.HasLen, 0)
	case <-time.After(testing.LongWait):
		c.Fatal("timed out waiting for event")
	}

	evt, ok := filter(provider.WatchEventUpdate, pod)
	c.Assert(ok, jc.IsTrue)
	podFirer([]string{evt})

	select {
	case v, ok := <-w.Changes():
		c.Assert(ok, jc.IsTrue)
		c.Assert(v, gc.DeepEquals, []string{"test-0"})
	case <-time.After(testing.LongWait):
		c.Fatal("timed out waiting for event")
	}
}

func (s *K8sBrokerSuite) TestWatchContainerStartDefaultWaitForUnit(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	podWatcher, podFirer := newKubernetesTestStringsWatcher()
	var filter provider.K8sStringsWatcherFilterFunc
	s.k8sStringsWatcherFn = provider.NewK8sStringsWatcherFunc(
		func(_ cache.SharedIndexInformer,
			_ string,
			_ jujuclock.Clock,
			_ []string,
			ff provider.K8sStringsWatcherFilterFunc) (provider.KubernetesStringsWatcher, error) {
			filter = ff
			return podWatcher, nil
		},
	)

	podList := &core.PodList{
		Items: []core.Pod{{
			ObjectMeta: v1.ObjectMeta{
				Name: "test-0",
				OwnerReferences: []v1.OwnerReference{
					{Kind: "StatefulSet"},
				},
			},
			Status: core.PodStatus{
				ContainerStatuses: []core.ContainerStatus{
					{Name: "first-container", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
				},
				Phase: core.PodPending,
			},
		}},
	}

	gomock.InOrder(
		s.mockPods.EXPECT().List(
			listOptionsLabelSelectorMatcher("juju-app==test"),
		).Return(podList, nil),
	)

	w, err := s.broker.WatchContainerStart("test", "")
	c.Assert(err, jc.ErrorIsNil)

	select {
	case v, ok := <-w.Changes():
		c.Assert(ok, jc.IsTrue)
		c.Assert(v, gc.HasLen, 0)
	case <-time.After(testing.LongWait):
		c.Fatal("timed out waiting for event")
	}

	pod := &core.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: "test-0",
			OwnerReferences: []v1.OwnerReference{
				{Kind: "StatefulSet"},
			},
			Annotations: map[string]string{
				"juju.io/unit": "test-0",
			},
		},
		Status: core.PodStatus{
			ContainerStatuses: []core.ContainerStatus{
				{Name: "first-container", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
			},
			Phase: core.PodPending,
		},
	}
	evt, ok := filter(provider.WatchEventUpdate, pod)
	c.Assert(ok, jc.IsTrue)
	podFirer([]string{evt})

	select {
	case v, ok := <-w.Changes():
		c.Assert(ok, jc.IsTrue)
		c.Assert(v, gc.DeepEquals, []string{"test-0"})
	case <-time.After(testing.LongWait):
		c.Fatal("timed out waiting for event")
	}
}

func (s *K8sBrokerSuite) TestUpgradeController(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	ss := apps.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Name: "controller",
			Annotations: map[string]string{
				"juju-version": "1.1.1",
			},
			Labels: map[string]string{"juju-operator": "controller"},
		},
		Spec: apps.StatefulSetSpec{
			Template: core.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{
						"juju-version": "1.1.1",
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{Image: "foo"},
						{Image: "jujud-operator:1.1.1"},
					},
				},
			},
		},
	}
	updated := ss
	updated.Annotations["juju-version"] = "6.6.6"
	updated.Spec.Template.Annotations["juju-version"] = "6.6.6"
	updated.Spec.Template.Spec.Containers[1].Image = "juju-operator:6.6.6"
	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("controller", v1.GetOptions{}).
			Return(&ss, nil),
		s.mockStatefulSets.EXPECT().Update(&updated).
			Return(nil, nil),
	)

	err := s.broker.Upgrade("controller", version.MustParse("6.6.6"))
	c.Assert(err, jc.ErrorIsNil)
}

func (s *K8sBrokerSuite) TestUpgradeNotSupported(c *gc.C) {
	ctrl := s.setupController(c)
	defer ctrl.Finish()

	gomock.InOrder(
		s.mockStatefulSets.EXPECT().Get("juju-operator-test-app", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockStatefulSets.EXPECT().Get("test-app-operator", v1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
	)

	err := s.broker.Upgrade("test-app", version.MustParse("6.6.6"))
	c.Assert(err, jc.Satisfies, errors.IsNotSupported)
}

func initContainers() []core.Container {
	jujudCmd := "export JUJU_DATA_DIR=/var/lib/juju\nexport JUJU_TOOLS_DIR=$JUJU_DATA_DIR/tools\n\nmkdir -p $JUJU_TOOLS_DIR\ncp /opt/jujud $JUJU_TOOLS_DIR/jujud"
	jujudCmd += `
initCmd=$($JUJU_TOOLS_DIR/jujud help commands | grep caas-unit-init)
if test -n "$initCmd"; then
$JUJU_TOOLS_DIR/jujud caas-unit-init --debug --wait;
else
exit 0
fi
`
	return []core.Container{{
		Name:            "juju-pod-init",
		Image:           "operator/image-path",
		Command:         []string{"/bin/sh"},
		Args:            []string{"-c", jujudCmd},
		WorkingDir:      "/var/lib/juju",
		VolumeMounts:    []core.VolumeMount{{Name: "juju-data-dir", MountPath: "/var/lib/juju"}},
		ImagePullPolicy: "IfNotPresent",
	}}
}

func dataVolumeMounts() []core.VolumeMount {
	return []core.VolumeMount{
		{
			Name:      "juju-data-dir",
			MountPath: "/var/lib/juju",
		},
		{
			Name:      "juju-data-dir",
			MountPath: "/usr/bin/juju-run",
			SubPath:   "tools/jujud",
		},
	}
}

func dataVolumes() []core.Volume {
	return []core.Volume{
		{
			Name: "juju-data-dir",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		},
	}
}
