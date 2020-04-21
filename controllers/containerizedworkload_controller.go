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
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cpv1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oamv1alpha2 "github.com/crossplane/crossplane/apis/oam/v1alpha2"
)

const (
	oamReconcileWait = 30 * time.Second
)

// Reconcile error strings.
const (
	errRenderWorkload  = "cannot render workload"
	errUpdateStatus    = "cannot apply status"
	errApplyDeployment = "cannot apply the deployment"
	errApplyService    = "cannot apply the service"
	errGCDeployment    = "cannot clean up stale deployments"
	errScaleDeployment = "cannot scale the deployment"
)

// ContainerizedWorkloadReconciler reconciles a ContainerizedWorkload object
type ContainerizedWorkloadReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *ContainerizedWorkloadReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("containerizedworkload", req.NamespacedName)
	log.Info("Reconcile container workload")

	var workload oamv1alpha2.ContainerizedWorkload
	if err := r.Get(ctx, req.NamespacedName, &workload); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Container workload is deleted")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Get the workload", "apiVersion", workload.APIVersion, "kind", workload.Kind)

	deploy, err := r.renderDeployment(ctx, &workload)
	if err != nil {
		workload.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errRenderWorkload)))
		log.Error(err, "Failed to render a deployment")
		return reconcile.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &workload),
			errUpdateStatus)
	}
	log.Info("Get a deployment", "deploy", deploy.Spec.Template.Spec.Containers[0])

	log.Info("Successfully rendered a deployment",
		"deployment name", deploy.Name,
		"deployment Namespace", deploy.Namespace,
		"number of containers", len(deploy.Spec.Template.Spec.Containers),
		"first container image", deploy.Spec.Template.Spec.Containers[0].Image)
	// server side apply, only the fields we set are touched
	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner(workload.Name)}
	if err := r.Patch(ctx, deploy, client.Apply, applyOpts...); err != nil {
		workload.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errApplyDeployment)))
		log.Error(err, "Failed to apply to a deployment")
		return reconcile.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &workload),
			errUpdateStatus)
	}
	log.Info("Successfully applied a deployment", "UID", deploy.UID)

	if err := r.Status().Update(ctx, &workload); err != nil {
		return reconcile.Result{RequeueAfter: oamReconcileWait}, err
	}

	// create a service for the workload
	// TODO(rz): Use ingress trait instead
	service, err := r.renderService(ctx, &workload, deploy)
	if err != nil {
		log.Error(err, "Failed to render a service")
		return reconcile.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &workload),
			errUpdateStatus)
	}
	// server side apply the service
	if err := r.Patch(ctx, service, client.Apply, applyOpts...); err != nil {
		workload.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errApplyService)))
		log.Error(err, "Failed to apply a service")
		return reconcile.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &workload),
			errUpdateStatus)
	}
	log.Info("Successfully applied a service", "UID", service.UID)

	// garbage collect the service/deployments that we created but not needed
	if err := r.cleanupResources(ctx, &workload, &deploy.UID, &service.UID); err != nil {
		workload.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errGCDeployment)))
		log.Error(err, "Failed to clean up resources")
		return reconcile.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &workload),
			errUpdateStatus)
	}
	workload.Status.Resources = nil
	// record the new deployment
	workload.Status.Resources = append(workload.Status.Resources, cpv1alpha1.TypedReference{
		APIVersion: deploy.GetObjectKind().GroupVersionKind().GroupVersion().String(),
		Kind:       deploy.GetObjectKind().GroupVersionKind().Kind,
		Name:       deploy.GetName(),
		UID:        deploy.UID,
	})
	// record the new service
	workload.Status.Resources = append(workload.Status.Resources, cpv1alpha1.TypedReference{
		APIVersion: service.GetObjectKind().GroupVersionKind().GroupVersion().String(),
		Kind:       service.GetObjectKind().GroupVersionKind().Kind,
		Name:       service.GetName(),
		UID:        service.UID,
	})

	if err := r.Status().Update(ctx, &workload); err != nil {
		return reconcile.Result{RequeueAfter: oamReconcileWait}, err
	}

	workload.Status.SetConditions(cpv1alpha1.ReconcileSuccess())
	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, &workload), errUpdateStatus)
}

func (r *ContainerizedWorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	src := &oamv1alpha2.ContainerizedWorkload{}
	return ctrl.NewControllerManagedBy(mgr).
		For(src).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
