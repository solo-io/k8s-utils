package kubeinstall_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestKubeinstall(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubeinstall Suite")
}
