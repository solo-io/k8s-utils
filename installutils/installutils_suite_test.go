package installutils_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInstallutils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installutils Suite")
}
