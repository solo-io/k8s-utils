package kube

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/solo-io/go-utils/testutils/kubectl"
)

type Image struct {
	Registry, Repository, Tag, Digest string
	PullPolicy                        string
}

func (i Image) String() string {
	b := strings.Builder{}
	b.WriteString(i.Registry)
	b.WriteRune('/')
	b.WriteString(i.Repository)
	if i.Tag != "" {
		b.WriteRune(':')
		b.WriteString(i.Tag)
	}
	if i.Digest != "" {
		b.WriteString("@sha256:")
		b.WriteString(i.Digest)
	}

	return b.String()
}

type EphemeralPodParams struct {
	Logger        io.Writer
	KubeContext   string
	Image         Image
	FromContainer string
	FromNamespace string
	FromPod       string
	ExecCmdPath   string
	Args          []string
	Env           []string
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
}

func ExecFromEphemeralPod(ctx context.Context, params EphemeralPodParams) (string, error) {
	createargs := []string{
		"debug",
		"--quiet",
		fmt.Sprintf("--image=%s", params.Image),
		fmt.Sprintf("--container=%s", params.FromContainer),
		fmt.Sprintf("--image-pull-policy=%s", params.Image.PullPolicy),
		params.FromPod,
		"-n", params.FromNamespace,
		"--", "sleep", "10h",
	}

	// create the params to the kubectl commands we will invoke.
	// we will use the same params but just switch out the args for the
	// different commands we execute.
	kParams := kubectl.NewParams(createargs...)
	kParams.Stdin = params.Stdin
	kParams.Stdout = params.Stdout
	kParams.Stderr = params.Stderr
	kParams.Env = params.Env

	// Execute curl commands from the same pod each time to avoid creating a burdensome number of ephemeral pods.
	// create the curl pod; we do this every time and it will only work the first time, so ignore failures
	_, _ = execute(ctx, KubectlParams{
		KubectlCmdParams: kParams,
		KubeContext:      params.KubeContext,
		Logger:           params.Logger,
	})

	// this smells; fix it
	// wait so the ephemeral container has a chance to be created before attempting to exec against it
	time.Sleep(time.Second)

	execArgs := []string{
		"exec",
		fmt.Sprintf("--container=%s", params.FromContainer),
		params.FromPod,
		"-n", params.FromNamespace,
		"--", params.ExecCmdPath,
	}

	kParams.Args = append(execArgs, params.Args...)
	return execute(ctx, KubectlParams{
		KubectlCmdParams: kParams,
		KubeContext:      params.KubeContext,
		Logger:           params.Logger,
	})
}
