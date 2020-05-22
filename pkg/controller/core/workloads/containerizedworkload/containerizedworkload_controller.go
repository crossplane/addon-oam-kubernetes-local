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

package containerizedworkload

import (
	"context"
	"strings"

	cpv1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/crossplane/oam-controllers/pkg/oam/util"
)

// Reconcile error strings.
const (
	errRenderWorkload  = "cannot render workload"
	errRenderService   = "cannot render service"
	errApplyDeployment = "cannot apply the deployment"
	errApplyService    = "cannot apply the service"
	errGCDeployment    = "cannot clean up stale deployments"
)

// Setup adds a controller that reconciles ContainerizedWorkload.
func Setup(mgr ctrl.Manager, log logging.Logger) error {
	reconciler := Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("ContainerizedWorkload"),
		Scheme: mgr.GetScheme(),
	}
	return reconciler.SetupWithManager(mgr)
}

// ContainerizedWorkloadReconciler reconciles a ContainerizedWorkload object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
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
		log.Error(err, "Failed to render a deployment")
		return util.ReconcileWaitResult,
			util.PatchCondition(ctx, r, &workload, cpv1alpha1.ReconcileError(errors.Wrap(err, errRenderWorkload)))
	}
	log.Info("Get a deployment", "deploy", deploy.Spec.Template.Spec.Containers[0])

	log.Info("Successfully rendered a deployment",
		"deployment name", deploy.Name,
		"deployment Namespace", deploy.Namespace,
		"number of containers", len(deploy.Spec.Template.Spec.Containers),
		"first container image", deploy.Spec.Template.Spec.Containers[0].Image)
	// server side apply, only the fields we set are touched
	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner(workload.GetUID())}
	if err := r.Patch(ctx, deploy, client.Apply, applyOpts...); err != nil {
		log.Error(err, "Failed to apply to a deployment")
		return util.ReconcileWaitResult,
			util.PatchCondition(ctx, r, &workload, cpv1alpha1.ReconcileError(errors.Wrap(err, errApplyDeployment)))
	}
	log.Info("Successfully applied a deployment", "UID", deploy.UID)

	// create a service for the workload
	// TODO(rz): Use ingress trait instead
	service, err := r.renderService(ctx, &workload, deploy)
	if err != nil {
		log.Error(err, "Failed to render a service")
		return util.ReconcileWaitResult,
			util.PatchCondition(ctx, r, &workload, cpv1alpha1.ReconcileError(errors.Wrap(err, errRenderService)))
	}
	// server side apply the service
	if err := r.Patch(ctx, service, client.Apply, applyOpts...); err != nil {
		log.Error(err, "Failed to apply a service")
		return util.ReconcileWaitResult,
			util.PatchCondition(ctx, r, &workload, cpv1alpha1.ReconcileError(errors.Wrap(err, errApplyService)))
	}
	log.Info("Successfully applied a service", "UID", service.UID)

	// garbage collect the service/deployments that we created but not needed
	if err := r.cleanupResources(ctx, &workload, &deploy.UID, &service.UID); err != nil {
		log.Error(err, "Failed to clean up resources")
		return util.ReconcileWaitResult,
			util.PatchCondition(ctx, r, &workload, cpv1alpha1.ReconcileError(errors.Wrap(err, errGCDeployment)))
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
		return util.ReconcileWaitResult, err
	}
	return ctrl.Result{}, util.PatchCondition(ctx, r, &workload, cpv1alpha1.ReconcileSuccess())
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	src := &oamv1alpha2.ContainerizedWorkload{}
	name := "oam/" + strings.ToLower(oamv1alpha2.ContainerizedWorkloadKind)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(src).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
