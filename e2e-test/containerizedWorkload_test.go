package controllers_test

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	oamv1alpha2 "github.com/crossplane/crossplane/apis/oam/v1alpha2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ContainerizedWorkload", func() {
	ctx := context.Background()
	namespace := "controller-test"
	lablel := map[string]string{"app": "test"}
	trueVar := true
	falseVar := false
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: lablel,
		},
	}
	BeforeEach(func() {
		logf.Log.Info("Start to run a test, clean up previous resources")
		// delete the namespace with all its resources
		Expect(k8sClient.Delete(ctx, &ns, client.PropagationPolicy(metav1.DeletePropagationForeground))).
			Should(SatisfyAny(BeNil(), &NotFoundMatcher{}))
		logf.Log.Info("make sure all the resources are removed")
		objectKey := client.ObjectKey{
			Name: namespace,
		}
		res := &corev1.Namespace{}
		Eventually(
			// gomega has a bug that can't take nil as the actual input, so has to make it a func
			func() error {
				return k8sClient.Get(ctx, objectKey, res)
			},
			time.Second*30, time.Millisecond*500).Should(&NotFoundMatcher{})
		// recreate it
		Eventually(
			// gomega has a bug that can't take nil as the actual input, so has to make it a func
			func() error {
				return k8sClient.Create(ctx, &ns)
			},
			time.Second*3, time.Millisecond*300).Should(SatisfyAny(BeNil(), &AlreadyExistMatcher{}))

	})
	AfterEach(func() {
		logf.Log.Info("Clean up resources")
		// delete the namespace with all its resources
		Expect(k8sClient.Delete(ctx, &ns, client.PropagationPolicy(metav1.DeletePropagationForeground))).Should(BeNil())
	})
	It("apply an application config", func() {
		wd := oamv1alpha2.WorkloadDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "containerizedworkloads.core.oam.dev",
				Labels: lablel,
			},
			Spec: oamv1alpha2.WorkloadDefinitionSpec{
				Reference: oamv1alpha2.DefinitionReference{
					Name: "containerizedworkloads.core.oam.dev",
				},
			},
		}
		logf.Log.Info("Creating workload definition")
		// For some reason, WorkloadDefinition is created as a Cluster scope object
		Expect(k8sClient.Create(ctx, &wd)).Should(SatisfyAny(BeNil(), &AlreadyExistMatcher{}))
		// Workload CR
		wl := oamv1alpha2.ContainerizedWorkload{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Labels:    lablel,
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ContainerizedWorkload",
			},
			Spec: oamv1alpha2.ContainerizedWorkloadSpec{
				Containers: []oamv1alpha2.Container{
					{
						Name:  "wordpress",
						Image: "wordpress:4.6.1-apache",
						Ports: []oamv1alpha2.ContainerPort{
							{
								Name: "wordpress",
								Port: 80,
							},
						},
					},
				},
			},
		}
		comp := oamv1alpha2.Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "example-component",
				Namespace: namespace,
				Labels:    lablel,
			},
			Spec: oamv1alpha2.ComponentSpec{
				Workload: runtime.RawExtension{
					Object: &wl,
				},
				Parameters: []oamv1alpha2.ComponentParameter{
					{
						Name:       "instance-name",
						Required:   &trueVar,
						FieldPaths: []string{"metadata.name"},
					},
					{
						Name:       "image",
						Required:   &falseVar,
						FieldPaths: []string{"spec.containers[0].image"},
					},
				},
			},
		}
		logf.Log.Info("Creating component", "Name", comp.Name, "Namespace", comp.Namespace)
		Expect(k8sClient.Create(ctx, &comp)).Should(BeNil())
		workloadName := "example-appconfig-workload"
		appConfig := oamv1alpha2.ApplicationConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "example-appconfig",
				Namespace: namespace,
				Labels:    lablel,
			},
			Spec: oamv1alpha2.ApplicationConfigurationSpec{
				Components: []oamv1alpha2.ApplicationConfigurationComponent{
					{
						ComponentName: "example-component",
						ParameterValues: []oamv1alpha2.ComponentParameterValue{
							{
								Name:  "instance-name",
								Value: intstr.IntOrString{StrVal: workloadName, Type: intstr.String},
							},
							{
								Name:  "image",
								Value: intstr.IntOrString{StrVal: "wordpress:php7.2", Type: intstr.String},
							},
						},
					},
				},
			},
		}
		logf.Log.Info("Creating application config", "Name", appConfig.Name, "Namespace", appConfig.Namespace)
		Expect(k8sClient.Create(ctx, &appConfig)).Should(BeNil())

		By("Checking deployment is created")
		objectKey := client.ObjectKey{
			Name:      workloadName,
			Namespace: namespace,
		}
		res := &appsv1.Deployment{}
		logf.Log.Info("Checking on deployment", "Key", objectKey)

		Eventually(
			func() error {
				return k8sClient.Get(ctx, objectKey, res)
			},
			time.Second*30, time.Millisecond*500).Should(BeNil())

		//Expect(len(res.Spec.Template.Spec.Containers[0].EnvFrom)).NotTo(BeZero())
	})
})
