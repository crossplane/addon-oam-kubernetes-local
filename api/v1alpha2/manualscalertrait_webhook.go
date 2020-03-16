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

package v1alpha2

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var manualscalertraitlog = logf.Log.WithName("manualscalertrait-resource")

func (r *ManualScalerTrait) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-manualscalertrait,mutating=true,failurePolicy=fail,groups=core.oam.dev,resources=manualscalertraits,verbs=create;update,versions=v1alpha2,name=manualscalertrait.mutate.core.oam.dev

var _ webhook.Defaulter = &ManualScalerTrait{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ManualScalerTrait) Default() {
	r.Spec.ReplicaCount = 5
	manualscalertraitlog.Info("set the ReplicaCount to 5", "scaler trait name", r.Name)
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-manualscalertrait,mutating=false,failurePolicy=fail,groups=core.oam.dev,resources=manualscalertraits,versions=v1alpha2,name=manualscalertrait.validate.core.oam.dev

var _ webhook.Validator = &ManualScalerTrait{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ManualScalerTrait) ValidateCreate() error {
	manualscalertraitlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ManualScalerTrait) ValidateUpdate(old runtime.Object) error {
	manualscalertraitlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ManualScalerTrait) ValidateDelete() error {
	manualscalertraitlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
