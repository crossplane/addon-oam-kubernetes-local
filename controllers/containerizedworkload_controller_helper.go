package controllers

import (
	"context"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oamv1alpha2 "github.com/oam-dev/core-resource-controller/api/v1alpha2"
)

const (
	OAMResourceTypeLabel = "oam.dev/type"
	OAMResourceNameLabel = "oam.dev/name"
	KindDeployment       = "Deployment"
	KindService          = "Service"
)

// OAMResourceTypes
type OAMResourceTypes string

// Supported OAM Resource Types
const (
	workloadType OAMResourceTypes = "workload"
)

// create a corresponding deployment
func (r *ContainerizedWorkloadReconciler) renderWorkload(ctx context.Context,
	workload *oamv1alpha2.ContainerizedWorkload) (*appsv1.Deployment, error) {
	var RevisionHistoryLimit int32 = 100
	deployName := workload.Name + "-deployment"
	depl := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       KindDeployment,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployName,
			Namespace: workload.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{OAMResourceTypeLabel: string(workloadType), OAMResourceNameLabel: deployName},
			},
			RevisionHistoryLimit: &RevisionHistoryLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{OAMResourceTypeLabel: string(workloadType), OAMResourceNameLabel: deployName},
				},
				Spec: corev1.PodSpec{
					Containers: workload.Spec.Containers,
				},
			},
		},
	}

	// always set the controller reference so that we can watch this deployment
	if err := ctrl.SetControllerReference(workload, &depl, r.Scheme); err != nil {
		return nil, err
	}

	return &depl, nil
}

// create a service for the deployment
func (r *ContainerizedWorkloadReconciler) renderService(ctx context.Context, deploy *appsv1.Deployment,
	port *corev1.ContainerPort) (*corev1.Service, error) {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       KindService,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploy.Name + "-service",
			Namespace: deploy.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       strings.ToLower(string(port.Protocol)),
					Port:       8080,
					Protocol:   port.Protocol,
					TargetPort: intstr.FromInt(int(port.ContainerPort)),
				},
			},
			Selector: map[string]string{OAMResourceNameLabel: deploy.Name},
			Type:     corev1.ServiceTypeNodePort,
		},
	}

	// always set the controller reference so that we can watch this service
	if err := ctrl.SetControllerReference(deploy, &svc, r.Scheme); err != nil {
		return nil, err
	}

	return &svc, nil
}

// delete deployments/services that are not the same as the existing
func (r *ContainerizedWorkloadReconciler) cleanupResources(ctx context.Context,
	workload *oamv1alpha2.ContainerizedWorkload, deployUID, serviceUID *types.UID) error {
	log := r.Log.WithValues("gc deployment", workload.Name)
	var deploy appsv1.Deployment
	var service corev1.Service
	for _, res := range workload.Status.Resources {
		uid := *res.UID
		if res.Kind == KindDeployment {
			if uid != *deployUID {
				log.Info("Found an orphaned deployment", "deployment UID", *deployUID, "orphaned  UID", uid)
				dn := client.ObjectKey{Name: res.Name, Namespace: workload.Namespace}
				if err := r.Get(ctx, dn, &deploy); err != nil {
					if apierrors.IsNotFound(err) {
						continue
					}
					return err
				}
				if err := r.Delete(ctx, &deploy); err != nil {
					return err
				}
				log.Info("Removed an orphaned deployment", "deployment UID", *deployUID, "orphaned UID", uid)
			}
		} else if res.Kind == KindService {
			if uid != *serviceUID {
				log.Info("Found an orphaned service", "orphaned  UID", uid)
				sn := client.ObjectKey{Name: res.Name, Namespace: workload.Namespace}
				if err := r.Get(ctx, sn, &service); err != nil {
					if apierrors.IsNotFound(err) {
						continue
					}
					return err
				}
				if err := r.Delete(ctx, &service); err != nil {
					return err
				}
				log.Info("Removed an orphaned service", "orphaned UID", uid)
			}
		}
	}
	return nil
}
