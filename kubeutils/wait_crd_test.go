package kubeutils_test

import (
	"context"

	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/solo-io/k8s-utils/kubeutils"
	apiexts "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("WaitCrd", func() {
	var (
		ctx     context.Context
		api     apiexts.Interface
		crdName = "testing"
	)
	BeforeEach(func() {
		ctx = context.Background()
		cfg, err := GetConfig("", "")
		Expect(err).NotTo(HaveOccurred())
		api, err = apiexts.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
		crd, err := api.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, &apiv1.CustomResourceDefinition{
			ObjectMeta: v1.ObjectMeta{Name: "somethings.test.solo.io"},
			Spec: apiv1.CustomResourceDefinitionSpec{
				Group: "test.solo.io",
				Names: apiv1.CustomResourceDefinitionNames{
					Plural:     "somethings",
					Kind:       "Something",
					ShortNames: []string{"st"},
				},
				Versions: []apiv1.CustomResourceDefinitionVersion{
					{
						Name: "v1",
					},
				},
			},
		}, v1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		crdName = crd.Name
	})
	AfterEach(func() {
		api.ApiextensionsV1().CustomResourceDefinitions().Delete(ctx, crdName, v1.DeleteOptions{})
	})
	It("waits successfully for a crd to become established", func() {
		err := WaitForCrdActive(ctx, api, crdName)
		Expect(err).NotTo(HaveOccurred())
	})
})
