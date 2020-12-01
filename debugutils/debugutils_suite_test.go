package debugutils

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/k8s-utils/installutils/helmchart"
	"github.com/solo-io/k8s-utils/installutils/kuberesource"
)

func TestDebugutils(t *testing.T) {
	T = t
	RegisterFailHandler(Fail)
	RunSpecs(t, "Debugutils Suite")
}

var (
	T    *testing.T
	ns   string
	ctrl *gomock.Controller

	manifests             helmchart.Manifests
	unstructuredResources kuberesource.UnstructuredResources

	_ = SynchronizedBeforeSuite(func() []byte {
		var err error
		manifests, err = helmchart.RenderManifests(
			context.Background(),
			"https://storage.googleapis.com/solo-public-helm/charts/gloo-0.13.33.tgz",
			"",
			"aaa",
			"gloo-system",
			"",
		)
		Expect(err).NotTo(HaveOccurred())
		unstructuredResources, err = manifests.ResourceList()
		Expect(err).NotTo(HaveOccurred())
		return nil
	}, func(data []byte) {})
)
