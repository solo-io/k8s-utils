package kuberesource_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestKuberesource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kuberesource Suite")
}


// setSoloClusterName via annotations onto the unstructured input
func setSoloClusterName(obj *unstructured.Unstructured){
	curAnnotations := obj.GetAnnotations()
	curAnnotation[]
}