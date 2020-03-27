/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhooks

import (
	"encoding/json"
	"fmt"

	logr "github.com/go-logr/logr"
	adminv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/crossplane/crossplane/apis/oam/v1alpha2"
)

type ManualScalerTraitValidator struct {
	Log logr.Logger
	gvk schema.GroupVersionKind
}

func (v ManualScalerTraitValidator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	convertToHttpHandler(v.validate, v.Log)(w, r)
}

// this is the default way, we will generate the path given gvk
func (v ManualScalerTraitValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	var err error
	v.gvk, err = apiutil.GVKForObject(&v1alpha2.ManualScalerTrait{}, mgr.GetScheme())
	if err != nil {
		return err
	}
	vPath := validate_path_prefix + generatePath(v.gvk)
	return RegisterWebhookWithManager(mgr, vPath, &v)
}

func (v ManualScalerTraitValidator) validate(ar adminv1.AdmissionReview) *adminv1.AdmissionResponse {
	log := v.Log.WithValues("name", ar.Request.Name, "Operation", ar.Request.Operation)
	log.Info("Start to validate a trait")
	expectedResource := metav1.GroupVersionResource{Group: "core.oam.dev", Version: "v1alpha2", Resource: "manualscalertraits"}

	if ar.Request.Resource != expectedResource {
		err := fmt.Errorf("wrong resource, expected %+v, got %+v ", expectedResource, ar.Request.Resource)
		log.Error(err, "expect resource mismatch")
		return toErrAdmissionResponse(err, http.StatusBadRequest)
	}

	raw := ar.Request.Object.Raw
	msTrait := v1alpha2.ManualScalerTrait{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &msTrait); err != nil {
		log.Error(err, "failed to decode")
		return toErrAdmissionResponse(err, http.StatusOK)
	}

	// validate trait spec
	if msTrait.Spec.ReplicaCount > 10 {
		log.Info("Maximum replica count exceed 10", "replicaCount", msTrait.Spec.ReplicaCount)
		return toErrAdmissionResponse(fmt.Errorf("maximum replica count 10, got %d", msTrait.Spec.ReplicaCount), http.StatusForbidden)

	}
	if len(msTrait.Spec.WorkloadReference.Name) == 0 ||
		len(msTrait.Spec.WorkloadReference.APIVersion) == 0 || len(msTrait.Spec.WorkloadReference.Kind) == 0 {
		log.Info("workload reference not valid", "replicaCount", msTrait.Spec.WorkloadReference)
		return toErrAdmissionResponse(fmt.Errorf("workload reference not valid, workload reference = %+v",
			msTrait.Spec.WorkloadReference), http.StatusForbidden)

	}
	return &adminv1.AdmissionResponse{
		Allowed: true,
		Result: &metav1.Status{
			Status: metav1.StatusSuccess,
		},
	}
}

type ManualScalerTraitMutater struct {
	Log logr.Logger
	gvk schema.GroupVersionKind
}

func (m ManualScalerTraitMutater) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	convertToHttpHandler(m.mutate, m.Log)(w, r)
}

// this is the default way, we will generate the path given gvk
func (m ManualScalerTraitMutater) SetupWebhookWithManager(mgr ctrl.Manager) error {
	var err error
	m.gvk, err = apiutil.GVKForObject(&v1alpha2.ManualScalerTrait{}, mgr.GetScheme())
	if err != nil {
		return err
	}
	vPath := mutate_path_prefix + generatePath(m.gvk)
	return RegisterWebhookWithManager(mgr, vPath, &m)
}

func (m ManualScalerTraitMutater) mutate(ar adminv1.AdmissionReview) *adminv1.AdmissionResponse {
	log := m.Log.WithValues("name", ar.Request.Name)
	log.Info("admitting manual scaler trait")
	expectedResource := metav1.GroupVersionResource{Group: "core.oam.dev", Version: "v1alpha2", Resource: "manualscalertraits"}

	if ar.Request.Resource != expectedResource {
		err := fmt.Errorf("wrong resource, expected %+v, got %+v ", expectedResource, ar.Request.Resource)
		log.Error(err, "expect resource mismatch")
		return toErrMutateResponse(err, http.StatusBadRequest)
	}

	raw := ar.Request.Object.Raw
	msTrait := v1alpha2.ManualScalerTrait{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &msTrait); err != nil {
		log.Error(err, "failed to decode")
		return toErrMutateResponse(err, http.StatusBadRequest)
	}

	if len(msTrait.Spec.WorkloadReference.Kind) == 0 {
		msTrait.Spec.WorkloadReference.Kind = "ContainerizedWorkload"
		log.Info("Default the WorkloadReference kind to ContainerizedWorkload")
	}
	if len(msTrait.Spec.WorkloadReference.APIVersion) == 0 {
		msTrait.Spec.WorkloadReference.APIVersion = msTrait.APIVersion
		log.Info("Default the WorkloadReference", "apiVersion", msTrait.APIVersion)
	}

	marshaledTrait, err := json.Marshal(msTrait)
	if err != nil {
		log.Error(err, "failed to marshal the mutated trait")
		return toErrMutateResponse(err, http.StatusInternalServerError)
	}
	// generate the patch using a lib
	patches, err := patchResponseFromRaw(ar.Request.Object.Raw, marshaledTrait)
	if err != nil {
		log.Error(err, "failed to generate the patch")
		return toErrMutateResponse(err, http.StatusInternalServerError)
	}
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		log.Error(err, "failed to marshal the patch")
		return toErrMutateResponse(err, http.StatusInternalServerError)
	}

	return &adminv1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		Result: &metav1.Status{
			Status: metav1.StatusSuccess,
		},
		PatchType: &pT,
		AuditAnnotations: map[string]string{
			"mutator": generatePath(m.gvk),
		},
	}
}
