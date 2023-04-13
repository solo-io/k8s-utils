package clusterlock_test

import (
	"context"
	"testing"

	"github.com/solo-io/go-utils/testutils/runners/consul"
	"github.com/solo-io/k8s-utils/testutils/clusterlock"
	"github.com/solo-io/k8s-utils/testutils/kube"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClusterlock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Clusterlock Suite")
}

var (
	consulFactory *consul.ConsulFactory
	kubeClient    kubernetes.Interface
)

var _ = BeforeSuite(func() {
	kubeClient = kube.MustKubeClient()
	var err error
	consulFactory, err = consul.NewConsulFactory()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	_ = consulFactory.Clean()
	kubeClient.CoreV1().ConfigMaps("default").Delete(context.Background(), clusterlock.LockResourceName, v1.DeleteOptions{})
})
