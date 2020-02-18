// Copyright 2019 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package exec_test

import (
	"bytes"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/juju/juju/caas/kubernetes/provider/exec"
	coretesting "github.com/juju/juju/testing"
)

func (s *execSuite) TestFileResourceValidate(c *gc.C) {
	ctrl := s.setupExecClient(c)
	defer ctrl.Finish()
	c.Assert((&exec.FileResource{}).Validate(), gc.ErrorMatches, `path was missing`)
}

func (s *execSuite) TestCopyParamsValidate(c *gc.C) {
	ctrl := s.setupExecClient(c)
	defer ctrl.Finish()

	type testcase struct {
		Params exec.CopyParams
		Err    string
	}
	for _, tc := range []testcase{
		{
			Params: exec.CopyParams{},
			Err:    "path was missing",
		},
		{
			Params: exec.CopyParams{
				Src: exec.FileResource{
					Path:    "",
					PodName: "",
				},
			},
			Err: "path was missing",
		},
		{
			Params: exec.CopyParams{
				Src: exec.FileResource{
					Path:    "/var/lib/juju/tools",
					PodName: "",
				},
				Dest: exec.FileResource{
					Path:    "",
					PodName: "",
				},
			},
			Err: "path was missing",
		},
		{
			Params: exec.CopyParams{
				Src: exec.FileResource{
					Path:    "/var/lib/juju/tools",
					PodName: "",
				},
				Dest: exec.FileResource{
					Path:    "/var/lib/juju/tools",
					PodName: "",
				},
			},
			Err: "copy either from pod to host or from host to pod",
		},
	} {
		c.Check(tc.Params.Validate(), gc.ErrorMatches, tc.Err)
	}

	// failed: can not copy from a pod to another pod.
	params := exec.CopyParams{
		Src: exec.FileResource{
			Path:    "/var/lib/juju/tools",
			PodName: "gitlab-k8s-0",
		},
		Dest: exec.FileResource{
			Path:    "/var/lib/juju/tools",
			PodName: "mariadb-k8s-0",
		},
	}
	c.Assert(params.Validate(), gc.ErrorMatches, "cross pods copy is not supported")
}

func (s *execSuite) TestCopyFromPodNotSupported(c *gc.C) {
	ctrl := s.setupExecClient(c)
	defer ctrl.Finish()

	cancel := make(chan struct{}, 1)

	params := exec.CopyParams{
		Src: exec.FileResource{
			Path:    "/var/lib/juju/tools",
			PodName: "gitlab-k8s-0",
		},
		Dest: exec.FileResource{
			Path:    "/var/lib/juju/tools",
			PodName: "",
		},
	}
	c.Assert(s.execClient.Copy(params, cancel), jc.Satisfies, errors.IsNotSupported)
}

func (s *execSuite) TestCopyToPod(c *gc.C) {
	ctrl := s.setupExecClient(c)
	defer ctrl.Finish()

	srcPath, err := ioutil.TempFile(c.MkDir(), "testfile")
	c.Assert(err, jc.ErrorIsNil)
	defer srcPath.Close()
	defer os.Remove(srcPath.Name())

	params := exec.CopyParams{
		Src: exec.FileResource{
			Path:    srcPath.Name(),
			PodName: "",
		},
		Dest: exec.FileResource{
			Path:    "/testdir",
			PodName: "gitlab-k8s-0",
		},
	}
	pod := core.Pod{
		Spec: core.PodSpec{
			Containers: []core.Container{
				{Name: "gitlab-container"},
			},
		},
		Status: core.PodStatus{
			Phase: core.PodRunning,
			ContainerStatuses: []core.ContainerStatus{
				{Name: "gitlab-container", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
			},
		},
	}
	pod.SetName("gitlab-k8s-0")

	checkRemotePathRequest := rest.NewRequestWithClient(
		&url.URL{Path: "/path/"},
		"",
		rest.ClientContentConfig{GroupVersion: core.SchemeGroupVersion},
		nil,
	).Resource("pods").Name("gitlab-k8s-0").Namespace("test").
		SubResource("exec").Param("container", "gitlab-container").VersionedParams(
		&core.PodExecOptions{
			Container: "gitlab-container",
			Command:   []string{"test", "-d", srcPath.Name()},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	copyRequest := rest.NewRequestWithClient(
		&url.URL{Path: "/path/"},
		"",
		rest.ClientContentConfig{GroupVersion: core.SchemeGroupVersion},
		nil,
	).Resource("pods").Name("gitlab-k8s-0").Namespace("test").
		SubResource("exec").Param("container", "gitlab-container").VersionedParams(
		&core.PodExecOptions{
			Container: "gitlab-container",
			Command:   []string{"tar", "-xmf", "-", "/testdir"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gomock.InOrder(
		// check remote path is dir or not.
		s.mockPodGetter.EXPECT().Get("gitlab-k8s-0", metav1.GetOptions{}).Return(&pod, nil),
		s.restClient.EXPECT().Post().Return(checkRemotePathRequest),
		s.mockRemoteCmdExecutor.EXPECT().Stream(
			remotecommand.StreamOptions{
				Stdout: &stdout,
				Stderr: &stderr,
				Tty:    false,
			},
		).Return(nil),

		// copy files.
		s.mockPodGetter.EXPECT().Get("gitlab-k8s-0", metav1.GetOptions{}).Return(&pod, nil),
		s.restClient.EXPECT().Post().Return(copyRequest),
		s.mockRemoteCmdExecutor.EXPECT().Stream(
			remotecommand.StreamOptions{
				Stdin:  s.pipReader,
				Stdout: &stdout,
				Stderr: &stderr,
				Tty:    false,
			},
		).Return(nil),
	)

	cancel := make(<-chan struct{}, 1)
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.execClient.Copy(params, cancel)
	}()
	select {
	case err := <-errChan:
		c.Assert(err, jc.ErrorIsNil)
	case <-time.After(coretesting.LongWait):
		c.Fatalf("timed out waiting for Copy return")
	}
}
