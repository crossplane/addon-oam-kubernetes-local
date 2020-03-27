package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oamv1alpha2 "github.com/crossplane/crossplane/apis/oam/v1alpha2"
)

const (
	KindDeployment = "Deployment"
	KindService    = "Service"
)

// OAMResourceTypes
type OAMResourceTypes string

// Supported OAM Resource Types
const (
	workloadType OAMResourceTypes = "workload"
)

// delete deployments/services that are not the same as the existing
func (r *ContainerizedWorkloadReconciler) cleanupResources(ctx context.Context,
	workload *oamv1alpha2.ContainerizedWorkload, deployUID, serviceUID *types.UID) error {
	log := r.Log.WithValues("gc deployment", workload.Name)
	var deploy appsv1.Deployment
	var service corev1.Service
	for _, res := range workload.Status.Resources {
		uid := res.UID
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
