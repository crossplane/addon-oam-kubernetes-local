package controllers

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	cpv1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// A generic status that includes an array of typedReference
type status struct {
	// Resources managed by this containerised workload.
	Resources []cpv1alpha1.TypedReference `json:"resources,omitempty"`
}

// A generic workload that has a status
type workload struct {
	Status status `json:"status"`
}

var _ = Describe("Manualscalertrait Controller Test", func() {
	BeforeEach(func() {
		logf.Log.Info("Set up resources before a unit test")
	})

	AfterEach(func() {
		logf.Log.Info("Clean up resources after a unit test")
	})

	It("Test extract Resources from any workload with status and type reference", func() {
		logf.Log.Info("Creating workload definition")
		genericStatus := status{
			Resources: []cpv1alpha1.TypedReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "deploy",
					UID:        "80d6b0f3-d543-480e-b7e3-cd648293ce80",
				},
				{
					APIVersion: "v1",
					Kind:       "Service",
					Name:       "service",
					UID:        "7d4b5579-fab3-45ba-8f0f-2263676637aa",
				},
			},
		}
		genericWorkload := workload{
			Status: genericStatus,
		}
		var object map[string]interface{}
		logf.Log.Info("Convert the workload structure into a map")
		workload, err := json.Marshal(genericWorkload)
		Expect(err).Should(BeNil())
		err = json.Unmarshal(workload, &object)
		Expect(err).Should(BeNil())
		unstrWorkload := unstructured.Unstructured{
			Object: object,
		}
		logf.Log.Info("Test unmarshal the map to typeReference array")
		refs, err := extractResources(unstrWorkload)
		Expect(err).Should(BeNil())
		for _, ref := range refs {
			if ref.Kind == "Deployment" {
				Expect(ref.APIVersion).Should(Equal("apps/v1"))
				Expect(ref.Name).Should(Equal("deploy"))
				Expect(ref.UID).Should(BeEquivalentTo("80d6b0f3-d543-480e-b7e3-cd648293ce80"))
			} else {
				Expect(ref.APIVersion).Should(Equal("v1"))
				Expect(ref.Kind).Should(Equal("Service"))
				Expect(ref.Name).Should(Equal("service"))
				Expect(ref.UID).Should(BeEquivalentTo("7d4b5579-fab3-45ba-8f0f-2263676637aa"))
			}
		}
	})
})
