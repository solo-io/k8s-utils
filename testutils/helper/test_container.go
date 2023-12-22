package helper

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/solo-io/go-utils/log"

	"github.com/solo-io/go-utils/testutils"
	"github.com/solo-io/k8s-utils/kubeutils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ TestRunner = &testRunner{}
var _ TestContainer = &testRunner{}
var _ TestContainer = &testContainer{}

// A TestContainer is a general-purpose abstraction over a container in which we might
// execute cURL or other, arbitrary commands via kubectl.
type TestContainer interface {
	Deploy(timeout time.Duration) error
	Terminate() error
	CanCurl() bool
	// Checks the response of the request
	CurlEventuallyShouldRespond(opts CurlOpts, substr string, ginkgoOffset int, timeout ...time.Duration)
	// Checks all of the output of the curl command
	CurlEventuallyShouldOutput(opts CurlOpts, substr string, ginkgoOffset int, timeout ...time.Duration)
	Curl(opts CurlOpts) (string, error)
	Exec(command ...string) (string, error)
	ExecAsync(args ...string) (io.Reader, chan struct{}, error)
}

// A TestRunner is an extension of a TestContainer which is typically run with the defaultTestRunnerImage
// and which has a service associated with it, and can run https.
type TestRunner interface {
	TestContainer
	DeployTLS(timeout time.Duration, crt, key []byte) error
	DeleteService() error
	TerminateAndDeleteService() error
	TestRunnerAsync(args ...string) (io.Reader, chan struct{}, error)
}

func newTestContainer(namespace, imageTag, echoName string, port int32) (*testContainer, error) {
	cfg, err := kubeutils.GetConfig("", "")
	if err != nil {
		return nil, err
	}
	kube, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &testContainer{
		namespace: namespace,
		kube:      kube,

		echoName: echoName,
		port:     port,
		imageTag: imageTag,
	}, nil
}

// This object represents a container that gets deployed to the cluster to support testing.
type testContainer struct {
	containerImageName string
	containerPort      uint
	namespace          string
	kube               kubernetes.Interface

	imageTag string
	echoName string
	port     int32
}

func (t *testContainer) Deploy(timeout time.Duration) error {
	return t.deploy(timeout)
}

// Deploys the http echo to the kubernetes cluster the kubeconfig is pointing to and waits for the given time for the
// http-echo pod to be running.
func (t *testContainer) deploy(timeout time.Duration) error {
	zero := int64(0)
	labels := map[string]string{"gloo": t.echoName}
	metadata := metav1.ObjectMeta{
		Name:      t.echoName,
		Namespace: t.namespace,
		Labels:    labels,
	}

	// Create http echo pod
	if _, err := t.kube.CoreV1().Pods(t.namespace).Create(context.TODO(), &corev1.Pod{
		ObjectMeta: metadata,
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &zero,
			Containers: []corev1.Container{
				{
					Image:           t.imageTag,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Name:            t.echoName,
				},
			},
		},
	}, metav1.CreateOptions{}); err != nil {
		return err
	}

	// Create http echo service
	if _, err := t.kube.CoreV1().Services(t.namespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metadata,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: corev1.ProtocolTCP,
					Port:     t.port,
				},
			},
			Selector: labels,
		},
	}, metav1.CreateOptions{}); err != nil {
		return err
	}

	// Wait until the http echo pod is running
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := testutils.WaitPodsRunning(ctx, time.Second, t.namespace, "gloo="+t.echoName); err != nil {
		return err
	}

	log.Printf("deployed %s", t.echoName)

	return nil
}

func (t *testContainer) Terminate() error {
	if err := testutils.Kubectl("delete", "pod", "-n", t.namespace, t.echoName, "--grace-period=0"); err != nil {
		return errors.Wrapf(err, "deleting %s pod", t.echoName)
	}
	return nil
}

func (t *testContainer) DeleteService() error {
	if err := testutils.Kubectl("delete", "service", "-n", t.namespace, t.echoName, "--grace-period=0"); err != nil {
		return errors.Wrapf(err, "deleting %s service", t.echoName)
	}
	return nil
}

func (t *testContainer) TerminateAndDeleteService() error {
	if err := t.Terminate(); err != nil {
		return err
	}
	if err := t.DeleteService(); err != nil {
		return err
	}
	return nil
}

// testContainer executes a command inside the testContainer container
func (t *testContainer) Exec(command ...string) (string, error) {
	args := append([]string{"exec", "-i", t.echoName, "-n", t.namespace, "--"}, command...)
	return testutils.KubectlOut(args...)
}

// Cp copies files into the testContainer container
func (t *testContainer) Cp(files map[string]string) error {
	for k, v := range files {
		if err := testutils.Kubectl("cp", k, fmt.Sprintf("%s/%s:%s", t.namespace, t.echoName, v)); err != nil {
			return err
		}
	}
	return nil
}

// TestRunnerAsync is deprecated; please use ExecAsync.
// TestContainerAsync executes a command inside the testContainer container
// returning a buffer that can be read from as it executes
func (t *testContainer) TestRunnerAsync(args ...string) (io.Reader, chan struct{}, error) {
	args = append([]string{"exec", "-i", t.echoName, "-n", t.namespace, "--"}, args...)
	return testutils.KubectlOutAsync(args...)
}

// ExecAsync executes a command inside the testContainer container
// returning a buffer that can be read from as it executes
func (t *testContainer) ExecAsync(args ...string) (io.Reader, chan struct{}, error) {
	args = append([]string{"exec", "-i", t.echoName, "-n", t.namespace, "--"}, args...)
	return testutils.KubectlOutAsync(args...)
}

func (t *testContainer) TestRunnerChan(r io.Reader, args ...string) (<-chan io.Reader, chan struct{}, error) {
	args = append([]string{"exec", "-i", t.echoName, "-n", t.namespace, "--"}, args...)
	return testutils.KubectlOutChan(r, args...)
}

func (t *testContainer) CanCurl() bool {
	if out, err := t.Exec("curl", "--version"); err != nil || !strings.HasPrefix(out, "curl") {
		return false
	}
	return true
}
