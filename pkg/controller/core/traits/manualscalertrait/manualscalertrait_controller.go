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

package manualscalertrait

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	cpv1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	cpmeta "github.com/crossplane/crossplane-runtime/pkg/meta"
	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/crossplane/oam-controllers/pkg/oam/util"
)

// Reconcile error strings.
const (
	oamReconcileWait = 30 * time.Second

	errLocateWorkload   = "cannot find workload"
	errLocateResources  = "cannot find resources"
	errLocateDeployment = "cannot find deployment"
	errUpdateStatus     = "cannot apply status"
	errScaleDeployment  = "cannot scale the deployment"
)

// Setup adds a controller that reconciles ContainerizedWorkload.
func Setup(mgr ctrl.Manager, log logging.Logger) error {
	reconciler := Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("ManualScalarTraitReconciler"),
		Scheme: mgr.GetScheme(),
	}
	return reconciler.SetupWithManager(mgr)
}

// Reconciler reconciles a ManualScalarTrait object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.oam.dev,resources=manualscalertraits,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.oam.dev,resources=manualscalertraits/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads,verbs=get;list;
// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads/status,verbs=get;
// +kubebuilder:rbac:groups=core.oam.dev,resources=workloaddefinition,verbs=get;list;
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch;delete

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	r.Log = r.Log.WithValues("manualscalar trait", req.NamespacedName)
	r.Log.Info("Reconcile manualscalar trait")

	var manualScalar oamv1alpha2.ManualScalerTrait
	if err := r.Get(ctx, req.NamespacedName, &manualScalar); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	r.Log.Info("Get the manualscalar trait", "ReplicaCount", manualScalar.Spec.ReplicaCount,
		"Annotations", manualScalar.GetAnnotations())

	// Fetch the workload instance this trait is referring to
	workload, result, err := r.fetchWorkload(ctx, &manualScalar)
	if err != nil {
		return result, err
	}

	// Fetch the child resources list from the corresponding workload
	resources, err := util.FetchWorkloadDefinition(ctx, r, workload)
	if err != nil {
		r.Log.Error(err, "Cannot find the workload child resources", "workload", workload.UnstructuredContent())
		manualScalar.Status.SetConditions(cpv1alpha1.ReconcileError(fmt.Errorf(errLocateResources)))
		return ctrl.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &manualScalar),
			errUpdateStatus)
	}

	// Scale the child resources we know how to scale
	result, err = r.scaleChildResources(ctx, manualScalar, resources)
	if err != nil {
		return result, err
	}

	manualScalar.Status.SetConditions(cpv1alpha1.ReconcileSuccess())

	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, &manualScalar), errUpdateStatus)
}

// TODO (rz): this is actually pretty generic, we can move this out into a common Trait structure with client and log
func (r *Reconciler) fetchWorkload(ctx context.Context,
	oamTrait oam.Trait) (*unstructured.Unstructured, ctrl.Result, error) {
	var workload unstructured.Unstructured
	workload.SetAPIVersion(oamTrait.GetWorkloadReference().APIVersion)
	workload.SetKind(oamTrait.GetWorkloadReference().Kind)
	wn := client.ObjectKey{Name: oamTrait.GetWorkloadReference().Name, Namespace: oamTrait.GetNamespace()}
	if err := r.Get(ctx, wn, &workload); err != nil {
		oamTrait.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errLocateWorkload)))
		r.Log.Error(err, "Workload not find", "kind", oamTrait.GetWorkloadReference().Kind,
			"workload name", oamTrait.GetWorkloadReference().Name)
		return nil, ctrl.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, oamTrait),
			errUpdateStatus)
	}
	r.Log.Info("Get the workload the trait is pointing to", "workload name", oamTrait.GetWorkloadReference().Name,
		"UID", workload.GetUID())
	return &workload, ctrl.Result{}, nil
}

// identify child resources and scale them
func (r *Reconciler) scaleChildResources(ctx context.Context, manualScalar oamv1alpha2.ManualScalerTrait,
	resources []*unstructured.Unstructured) (ctrl.Result, error) {
	// scale all the child resources that is of kind deployment
	isController := false
	bod := true
	found := false
	// Update owner references
	ref := metav1.OwnerReference{
		APIVersion:         manualScalar.APIVersion,
		Kind:               manualScalar.Kind,
		Name:               manualScalar.Name,
		UID:                manualScalar.UID,
		Controller:         &isController,
		BlockOwnerDeletion: &bod,
	}
	for _, res := range resources {
		if res.GetKind() == util.KindDeployment && res.GetAPIVersion() == appsv1.SchemeGroupVersion.String() {
			found = true
			r.Log.Info("Get the deployment the trait is going to modify",
				"deploy name", res.GetName(), "UID", res.GetUID())
			// convert the unstructured to deployment and scale replica
			var sd appsv1.Deployment
			bts, _ := json.Marshal(res)
			if err := json.Unmarshal(bts, &sd); err != nil {
				r.Log.Error(err, "Failed to convert an unstructured obj to a deployment")
				continue
			}
			cpmeta.AddOwnerReference(&sd, ref)
			sd.Spec.Replicas = &manualScalar.Spec.ReplicaCount
			// merge to scale the deployment
			if err := r.Patch(ctx, &sd, client.MergeFrom(res)); err != nil {
				manualScalar.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errScaleDeployment)))
				r.Log.Error(err, "Failed to scale a deployment")
				return reconcile.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &manualScalar),
					errUpdateStatus)
			}
			r.Log.Info("Successfully scaled a deployment", "UID", sd.GetUID(), "target replica",
				manualScalar.Spec.ReplicaCount)
		}
	}
	if !found {
		r.Log.Info("Cannot locate any deployment", "total resources", len(resources))
		manualScalar.Status.SetConditions(cpv1alpha1.ReconcileError(fmt.Errorf(errLocateDeployment)))
		return ctrl.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &manualScalar),
			errUpdateStatus)
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&oamv1alpha2.ManualScalerTrait{}).
		Watches(&source.Kind{
			Type: &appsv1.Deployment{},
		}, &handler.EnqueueRequestForOwner{
			OwnerType:    &oamv1alpha2.ManualScalerTrait{},
			IsController: false, // we only added a owner reference to it as there can only be one
		}).
		Complete(r)
}
