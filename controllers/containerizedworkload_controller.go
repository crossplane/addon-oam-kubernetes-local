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
	"time"

	cpv1alpha1 "github.com/crossplaneio/crossplane-runtime/apis/core/v1alpha1"
	//cplogging "github.com/crossplaneio/crossplane-runtime/pkg/logging"
	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	oamv1alpha2 "github.com/oam-dev/core-resource-controller/api/v1alpha2"
)

const (
	oamReconcileWait = 30 * time.Second
)

// Reconcile error strings.
const (
	errRenderWorkload   = "cannot render workload"
	errUpdateStatus     = "cannot apply status"
	errApplyDeployment  = "cannot apply the deployment"
	errApplyService     = "cannot apply the service"
	errGCDeployment     = "cannot clean up stale deployments"
	errUpdateDeployment = "cannot update the deployment"
	errScaleDeployment  = "cannot scale the deployment"
)

// ContainerizedWorkloadReconciler reconciles a ContainerizedWorkload object
type ContainerizedWorkloadReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads/status,verbs=get;update;patch

func (r *ContainerizedWorkloadReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("containerizedworkload", req.NamespacedName)
	log.Info("Reconcile container workload")

	var workload oamv1alpha2.ContainerizedWorkload
	if err := r.Get(ctx, req.NamespacedName, &workload); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Get the workload")

	deploy, err := r.renderWorkload(ctx, &workload)
	if err != nil {
		workload.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errRenderWorkload)))
		log.Error(err, "Failed to render a deployment")
		return reconcile.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &workload),
			errUpdateStatus)
	}
	log.Info("Successfully rendered a deployment", "deployment", deploy.Name)

	// server side apply, only the fields we set are touched
	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner(workload.ObjectMeta.Name)}
	if err := r.Patch(ctx, deploy, client.Apply, applyOpts...); err != nil {
		workload.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errApplyDeployment)))
		log.Error(err, "Failed to apply to a deployment")
		return reconcile.Result{RequeueAfter: oamReconcileWait}, errors.Wrap(r.Status().Update(ctx, &workload),
			errUpdateStatus)
	}
	log.Info("Successfully applied a deployment", "UID", deploy.UID)

	// TODO(rz): Use ingress trait instead
	// set up a service for the deployment if one of the containers have port
	var service *corev1.Service
	for _, c := range deploy.Spec.Template.Spec.Containers {
		// TODO (rz): pass in all the ports
		if len(c.Ports) != 0 {
			service, err = r.renderService(ctx, deploy, &c.Ports[0])
			if err != nil {
				log.Error(err, "Failed to render a service")
				continue
			}
			break
		}
	}
	// always set the controller reference so that we can watch this service
	if err := ctrl.SetControllerReference(&workload, service, r.Scheme); err != nil {
		workload.Status.SetConditions(cpv1alpha1.ReconcileError(errors.Wrap(err, errApplyService)))
		log.Error(err, "Failed to set the controller owner reference for a service")
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
	workload.Status.Resources = append(workload.Status.Resources, oamv1alpha2.ResourceReference{
		APIVersion: deploy.APIVersion,
		Kind:       deploy.Kind,
		Name:       deploy.Name,
		UID:        &deploy.UID,
	})
	// record the new service
	workload.Status.Resources = append(workload.Status.Resources, oamv1alpha2.ResourceReference{
		APIVersion: service.APIVersion,
		Kind:       service.Kind,
		Name:       service.Name,
		UID:        &service.UID,
	})

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
