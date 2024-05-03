package kubeutils

import (
	"crypto/md5"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gmeasure"
)

// Here for fuzz tests..
func sanitizeNameV2Old(name string) string {
	name = strings.Replace(name, "*", "-", -1)
	name = strings.Replace(name, "/", "-", -1)
	name = strings.Replace(name, ".", "-", -1)
	name = strings.Replace(name, "[", "", -1)
	name = strings.Replace(name, "]", "", -1)
	name = strings.Replace(name, ":", "-", -1)
	name = strings.Replace(name, "_", "-", -1)
	name = strings.Replace(name, " ", "-", -1)
	name = strings.Replace(name, "\n", "", -1)
	name = strings.Replace(name, "\"", "", -1)
	name = strings.Replace(name, "'", "", -1)
	if len(name) > 63 {
		hash := md5.Sum([]byte(name))
		name = fmt.Sprintf("%s-%x", name[:31], hash)
		name = name[:63]
	}
	name = strings.Replace(name, ".", "-", -1)
	name = strings.ToLower(name)
	return name
}

var _ = Describe("sanitize name", func() {

	DescribeTable("sanitize short names", func(in, out string) {
		Expect(SanitizeNameV2(in)).To(Equal(out))
	},
		Entry("basic a", "abc", "abc"),
		Entry("basic A", "Abc", "abc"),
		Entry("basic b", "abc123", "abc123"),
		Entry("subX *", "bb*", "bb-"),
		Entry("sub *", "bb*b", "bb-b"),
		Entry("subX /", "bb/", "bb-"),
		Entry("sub /", "bb/b", "bb-b"),
		Entry("subX .", "bb.", "bb-"),
		Entry("sub .", "bb.b", "bb-b"),
		Entry("sub0 [", "bb[", "bb"),
		Entry("sub [", "bb[b", "bbb"),
		Entry("sub0 ]", "bb]", "bb"),
		Entry("sub ]", "bb]b", "bbb"),
		Entry("subX :", "bb:", "bb-"),
		Entry("sub :", "bb:b", "bb-b"),
		Entry("subX space", "bb ", "bb-"),
		Entry("sub space", "bb b", "bb-b"),
		Entry("subX newline", "bb\n", "bb"),
		Entry("sub newline", "bb\nb", "bbb"),
		Entry("sub0 quote", "aa\"", "aa"),
		Entry("sub quote b", "bb\"b", "bbb"),
		Entry("sub0 single quote", "aa'", "aa"),
		Entry("sub single quote b", "bb'b", "bbb"),
		// these are technically invalid kube names, as are the subX cases, but user should know that and kube wil warn
		Entry("invalid a", "123", "123"),
		Entry("invalid b", "-abc", "-abc"),
	)

	DescribeTable("sanitize long names", func(in, out string) {
		sanitized := SanitizeNameV2(in)
		Expect(sanitized).To(Equal(out))
		Expect(len(sanitized)).To(BeNumerically("<=", 63))
	},
		Entry("300a's", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-4e5475d125a33c6190718e75adc1b70"),
		Entry("301a's", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-c73301b7b71679067b02cff4cdc5e70"),
	)
	It("SanitizeNameV2 efficiently", Serial, Label("measurement"), func() {
		experiment := gmeasure.NewExperiment("Repaginating Books")
		AddReportEntry(experiment.Name, experiment)

		experiment.Sample(func(idx int) {
			experiment.MeasureDuration("repagination", func() {
				for i := 0; i < 1000; i++ {
					SanitizeNameV2("sub []_---]_9da02_--_2")
				}
			})
		}, gmeasure.SamplingConfig{N: 200, Duration: time.Minute})
	})
})

func FuzzSanitizeNameParity(f *testing.F) {
	// Random string < 63
	f.Add("VirtualGateway-istio-ingressgateway-bookinfo-cluster-1-istio-ingressgateway-istio-gateway-ns-cluster-1-gloo-mesh-cluster-1-HTTPS.443-anything")
	f.Add("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	f.Add("abc")
	f.Add("abc123")
	f.Add("bb*")
	f.Add("bb*b")
	f.Add("bb/")
	f.Add("bb/b")
	f.Add("bb.")
	f.Add("bb.b")
	f.Add("bb[")
	f.Add("bb[b")
	f.Add("bb]")
	f.Add("bb]b")
	f.Add("bb:")
	f.Add("bb:b")
	f.Add("bb ")
	f.Add("bb b")
	f.Add("bb\n")
	f.Add("bb\nb")
	f.Add("aa\"")
	f.Add("bb\"b")
	f.Add("aa'")
	f.Add("bb'b")
	f.Add("jfdklanfkljasfhjhldacaslkhdfkjshfkjsadhfkjasdhgjadhgkdahfjkdahjfdsagdfhjdsagfhasjdfsdfasfsafsdf")

	f.Fuzz(func(t *testing.T, a string) {
		// we can only  get a valid kube name that's alphanumeric
		if !utf8.Valid([]byte(a)) {
			t.Skip("Skipping non-valid utf8 input")
		}
		oldName := SanitizeNameV2(a)
		newName := sanitizeNameV2Old(a)
		if oldName != newName {
			t.Fatalf("SanitizeNameV2(%s) = %s, SanitizeNameV2Old(%s) = %s", a, oldName, a, newName)
		}
	})
}
