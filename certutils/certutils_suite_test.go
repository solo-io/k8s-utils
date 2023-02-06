package certutils_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCertutils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Certutils Suite")
}
