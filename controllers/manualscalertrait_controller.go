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

package controllers

import (
	"context"
	"fmt"
	cpv1alpha1 "github.com/crossplaneio/crossplane-runtime/apis/core/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	oamv1alpha2 "github.com/oam-dev/core-resource-controller/api/v1alpha2"
)

// Reconcile error strings.
const (
	errLocateWorkload   = "cannot find workload"
	errLocateDeployment = "cannot find deployment"
)

// ManualScalerTraitReconciler reconciles a ManualScalerTrait object
type ManualScalerTraitReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.oam.dev,resources=manualscalertraits,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.oam.dev,resources=manualscalertraits/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads,verbs=get;list;
// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads/status,verbs=get;
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch;delete

func (r *ManualScalerTraitReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("manualscaler trait", req.NamespacedName)
	log.Info("Reconcile manualscalar trait")

	var manualScaler oamv1alpha2.ManualScalerTrait
	if err := r.Get(ctx, req.NamespacedName, &manualScaler); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Get the manualscaler trait", "ReplicaCount", manualScaler.Spec.ReplicaCount,
		"WorkloadReference", manualScaler.Spec.WorkloadReference)

	// Fetch the workload this trait is referring to
	var workload oamv1alpha2.ContainerizedWorkload
	wn := client.ObjectKey{Name: manualScaler.Spec.WorkloadReference.Name, Namespace: req.Namespace}
	if err := r.Get(ctx, wn, &workload); err != nil {
		manualScaler.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errLocateWorkload)))
		return ctrl.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &manualScaler),
			errUpdateStatus)
	}
	log.Info("Get the workload the trait is pointing to", "workload name", manualScaler.Spec.WorkloadReference.Name,
		"UID", workload.UID)

	if manualScaler.Spec.WorkloadReference.UID == nil || workload.UID != *manualScaler.Spec.WorkloadReference.UID {
		log.Info("Wrong workload", "trait references to ", manualScaler.Spec.WorkloadReference.UID)
		manualScaler.Status.SetConditions(cpv1alpha1.ReconcileError(fmt.Errorf(errLocateWorkload)))
		return ctrl.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &manualScaler),
			errUpdateStatus)
	}

	// TODO(rz): only apply if there is only one deployment
	// Fetch the deployment we are going to modify
	var scaleDeploy appsv1.Deployment
	found := false
	for _, res := range workload.Status.Resources {
		if res.Kind == KindDeployment {
			dn := client.ObjectKey{Name: res.Name, Namespace: req.Namespace}
			if err := r.Get(ctx, dn, &scaleDeploy); err != nil {
				log.Error(err, "Failed to get an associated deployment", "name ", res.Name)
				manualScaler.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errLocateDeployment)))
				continue
			}
			found = true
			break
		}
	}
	if !found {
		log.Info("Cannot locate a deployment", "total resources", len(workload.Status.Resources))
		manualScaler.Status.SetConditions(cpv1alpha1.ReconcileError(fmt.Errorf(errLocateDeployment)))
		return ctrl.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &manualScaler),
			errUpdateStatus)
	}
	log.Info("Get the deployment the trait is going to modify", "deploy name", scaleDeploy.Name, "UID", scaleDeploy.UID)

	sd := scaleDeploy.DeepCopy()
	// always set the owner reference so that we can watch this deployment
	isController := false
	bod := true
	// Create a new owner ref
	ref := metav1.OwnerReference{
		APIVersion:         manualScaler.APIVersion,
		Kind:               manualScaler.Kind,
		Name:               manualScaler.Name,
		UID:                manualScaler.UID,
		Controller:         &isController,
		BlockOwnerDeletion: &bod,
	}

	existingRefs := scaleDeploy.GetOwnerReferences()
	fi := -1
	for i, r := range existingRefs {
		if r.UID == manualScaler.UID {
			fi = i
			break
		}
	}
	if fi == -1 {
		existingRefs = append(existingRefs, ref)
	} else {
		existingRefs[fi] = ref
	}
	// Update owner references
	scaleDeploy.SetOwnerReferences(existingRefs)
	// merge to scale the deployment
	if err := r.Patch(ctx, sd, client.MergeFrom(&scaleDeploy)); err != nil {
		manualScaler.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errScaleDeployment)))
		log.Error(err, "Failed to scale a deployment")
		return reconcile.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &manualScaler),
			errUpdateStatus)
	}
	log.Info("Successfully scaled a deployment", "UID", scaleDeploy.UID)
	manualScaler.Status.SetConditions(cpv1alpha1.ReconcileSuccess())
	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, &manualScaler), errUpdateStatus)
}

func (r *ManualScalerTraitReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
