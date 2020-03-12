package controllers

import (
	"context"
	oamv1alpha2 "github.com/oam-dev/core-resource-controller/api/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"testing"
)

func TestContainerizedWorkloadReconciler_cleanupResources(t *testing.T) {
	type args struct {
		ctx        context.Context
		workload   *oamv1alpha2.ContainerizedWorkload
		deployUID  *types.UID
		serviceUID *types.UID
	}
	testCases := map[string]struct {
		reconciler ContainerizedWorkloadReconciler
		args       args
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := testCase.reconciler.cleanupResources(testCase.args.ctx, testCase.args.workload, testCase.args.deployUID,
				testCase.args.serviceUID); (err != nil) != testCase.wantErr {
				t.Errorf("cleanupResources() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestContainerizedWorkloadReconciler_renderService(t *testing.T) {
	type args struct {
		ctx      context.Context
		deploy   *appsv1.Deployment
		workload *oamv1alpha2.ContainerizedWorkload
	}
	testCases := map[string]struct {
		reconciler ContainerizedWorkloadReconciler
		args       args
		want       *corev1.Service
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := testCase.reconciler.renderService(testCase.args.ctx, testCase.args.deploy, testCase.args.workload)
			if (err != nil) != testCase.wantErr {
				t.Errorf("renderService() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("renderService() got = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestContainerizedWorkloadReconciler_renderWorkload(t *testing.T) {
	type args struct {
		ctx      context.Context
		workload *oamv1alpha2.ContainerizedWorkload
	}
	testCases := map[string]struct {
		reconciler ContainerizedWorkloadReconciler
		args       args
		want       *appsv1.Deployment
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := testCase.reconciler.renderWorkload(testCase.args.ctx, testCase.args.workload)
			if (err != nil) != testCase.wantErr {
				t.Errorf("renderWorkload() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("renderWorkload() got = %v, want %v", got, testCase.want)
			}
		})
	}
}
