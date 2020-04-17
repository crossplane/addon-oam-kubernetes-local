package controllers

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	"k8s.io/apimachinery/pkg/types"

	oamv1alpha2 "github.com/crossplane/crossplane/apis/oam/v1alpha2"
)

var _ = Describe("Manualscalertrait Controller Test", func() {

})

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
