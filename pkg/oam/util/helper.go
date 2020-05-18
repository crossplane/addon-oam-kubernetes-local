package util

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	plur "github.com/gertd/go-pluralize"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
)

const (
	KindDeployment = "Deployment"
	KindService    = "Service"
)

// FetchWorkloadDefinition fetch corresponding workloadDefinition given a workload
func FetchWorkloadDefinition(ctx context.Context, r client.Reader, workload *unstructured.Unstructured) (
	[]*unstructured.Unstructured, error) {
	// The name of the workloadDefinition CR is the CRD name of the component which is <purals>.<group>
	gvr := getGVResource(workload.Object)
	wldName := gvr.Resource + "." + gvr.Group
	nn := types.NamespacedName{Name: wldName}
	// Fetch the corresponding workloadDefinition CR
	workloadDefinition := &v1alpha2.WorkloadDefinition{}
	if err := r.Get(ctx, nn, workloadDefinition); err != nil {
		return nil, err
	}
	return fetchChildResources(ctx, r, workload, workloadDefinition.Spec.ChildResourceKinds)
}

func fetchChildResources(ctx context.Context, r client.Reader, workload *unstructured.Unstructured,
	wcrl []v1alpha2.ChildResourceKind) ([]*unstructured.Unstructured, error) {
	var childResources []*unstructured.Unstructured
	// list by each child resource type with namespace and possible label selector
	for _, wcr := range wcrl {
		crs := unstructured.UnstructuredList{}
		crs.SetAPIVersion(wcr.APIVersion)
		crs.SetKind(wcr.Kind)
		if err := r.List(ctx, &crs, client.InNamespace(workload.GetNamespace()),
			client.MatchingLabels(wcr.Selector)); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to list object %s.%s", crs.GetAPIVersion(), crs.GetKind()))
		}
		// pick the ones that is owned by the workload
		for _, cr := range crs.Items {
			for _, owner := range cr.GetOwnerReferences() {
				if owner.UID == workload.GetUID() {
					childResources = append(childResources, &cr)
				}
			}
		}
	}
	return childResources, nil
}

func getGVResource(ob map[string]interface{}) metav1.GroupVersionResource {
	apiVersion, _, _ := unstructured.NestedString(ob, "apiVersion")
	kind, _, _ := unstructured.NestedString(ob, "kind")
	g, v := ApiVersion2GroupVersion(apiVersion)
	return metav1.GroupVersionResource{
		Group:    g,
		Version:  v,
		Resource: Kind2Resource(kind),
	}
}

// ApiVersion2GroupVersion turn an apiVersion string into group and version
func ApiVersion2GroupVersion(str string) (string, string) {
	strs := strings.Split(str, "/")
	if len(strs) == 2 {
		return strs[0], strs[1]
	}
	// core type
	return "", strs[0]
}

// Kind2Resource convert Kind to Resources
func Kind2Resource(str string) string {
	return plur.NewClient().Plural(strings.ToLower(str))
}

// Object2Unstructured convert an object to an unstructured struct
func Object2Unstructured(obj interface{}) (*unstructured.Unstructured, error) {
	objMap, err := Object2Map(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{
		Object: objMap,
	}, nil
}

// Object2Map turn the Object to a map
func Object2Map(obj interface{}) (map[string]interface{}, error) {
	var res map[string]interface{}
	bts, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bts, &res)
	return res, err
}
