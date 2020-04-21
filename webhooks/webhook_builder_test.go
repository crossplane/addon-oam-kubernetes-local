package webhooks

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("Webhook builder unit test", func() {
	BeforeEach(func() {

	})
	AfterEach(func() {
	})
	It("generatePath test", func() {
		testCases := map[string]struct {
			gvk    schema.GroupVersionKind
			result string
		}{
			"base line case ": {
				gvk: schema.GroupVersionKind{
					Group:   "k8s.io",
					Version: "v1alpha",
					Kind:    "test",
				},
				result: "k8s-io-v1alpha-test",
			},
			"lower case kind case ": {
				gvk: schema.GroupVersionKind{
					Group:   "k8s.io",
					Version: "v1alpha",
					Kind:    "Test",
				},
				result: "k8s-io-v1alpha-test",
			},
			"dot replace case kind case ": {
				gvk: schema.GroupVersionKind{
					Group:   "sig.k8s.io",
					Version: "v1alpha",
					Kind:    "Test",
				},
				result: "sig-k8s-io-v1alpha-test",
			},
		}
		for name, testCase := range testCases {
			logf.Log.Info("Start to run test", "test", name)
			Expect(generatePath(testCase.gvk)).Should(BeIdenticalTo(testCase.result))
		}
	})
})
