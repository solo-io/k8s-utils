package kubeutils_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestKubeutils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubeutils Suite")
}
