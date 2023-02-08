package nameutils_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNameutils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nameutils Suite")
}
