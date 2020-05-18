package util_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/crossplane/oam-controllers/pkg/oam/util"
)

var _ = Describe("Test helper utils", func() {
	// Test common variables
	ctx := context.Background()
	namespace := "oamNS"
	workloadName := "oamWorkload"
	workloadKind := "ContainerizedWorkload"
	workloadAPIVersion := "core.oam.dev/v1"
	workloadDefinitionName := "containerizedworkloads.core.oam.dev"
	var workloadUID types.UID = "oamWorkloadUID"
	// workload CR
	workload := v1alpha2.ContainerizedWorkload{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadName,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: workloadAPIVersion,
			Kind:       workloadKind,
		},
	}
	workload.SetUID(workloadUID)
	unstructuredWorkload, _ := util.Object2Unstructured(workload)
	// deployment resources pointing to the workload
	deployment := unstructured.Unstructured{}
	deployment.SetOwnerReferences([]metav1.OwnerReference{
		{
			UID: workloadUID,
		},
	})
	// workload Definition
	workloadDefinition := v1alpha2.WorkloadDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: workloadDefinitionName,
		},
		Spec: v1alpha2.WorkloadDefinitionSpec{
			Reference: v1alpha2.DefinitionReference{
				Name: workloadDefinitionName,
			},
		},
	}
	crkl := []v1alpha2.ChildResourceKind{
		{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
	}
	getErr := fmt.Errorf("get failed")

	var nilListFunc test.ObjectFn = func(o runtime.Object) error {
		u := &unstructured.Unstructured{}
		l := o.(*unstructured.UnstructuredList)
		l.Items = []unstructured.Unstructured{*u}
		return nil
	}

	BeforeEach(func() {
		logf.Log.Info("Set up resources before a unit test")
	})

	AfterEach(func() {
		logf.Log.Info("Clean up resources after a unit test")
	})

	It("Test extract child resources from any workload", func() {
		type fields struct {
			getFunc  test.ObjectFn
			listFunc test.ObjectFn
		}
		type want struct {
			crks []*unstructured.Unstructured
			err  error
		}

		cases := map[string]struct {
			fields fields
			want   want
		}{
			"FetchWorkloadDefinition fail when getWorkloadDefinition fails": {
				fields: fields{
					getFunc: func(obj runtime.Object) error {
						return getErr
					},
					listFunc: nilListFunc,
				},
				want: want{
					crks: nil,
					err:  getErr,
				},
			},
			"FetchWorkloadDefinition return nothing when the workloadDefinition doesn't have child list": {
				fields: fields{
					getFunc: func(obj runtime.Object) error {
						o, _ := obj.(*v1alpha2.WorkloadDefinition)
						*o = workloadDefinition
						return nil
					},
					listFunc: nilListFunc,
				},
				want: want{
					crks: nil,
					err:  nil,
				},
			},
			"FetchWorkloadDefinition Success": {
				fields: fields{
					getFunc: func(obj runtime.Object) error {
						o, _ := obj.(*v1alpha2.WorkloadDefinition)
						w := workloadDefinition
						w.Spec.ChildResourceKinds = crkl
						*o = w
						return nil
					},
					listFunc: func(o runtime.Object) error {
						l := o.(*unstructured.UnstructuredList)
						if l.GetKind() != util.KindDeployment {
							return getErr
						}
						l.Items = []unstructured.Unstructured{deployment}
						return nil
					},
				},
				want: want{
					crks: []*unstructured.Unstructured{
						&deployment,
					},
					err: nil,
				},
			},
		}
		for name, tc := range cases {
			tclient := test.MockClient{
				MockGet:  test.NewMockGetFn(nil, tc.fields.getFunc),
				MockList: test.NewMockListFn(nil, tc.fields.listFunc),
			}
			got, err := util.FetchWorkloadDefinition(ctx, &tclient, unstructuredWorkload)
			By(fmt.Sprint("Running test: ", name))
			Expect(tc.want.err).Should(util.BeEquivalentToError(err))
			Expect(tc.want.crks).Should(Equal(got))
		}
	})
})
