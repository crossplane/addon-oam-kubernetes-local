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

package v1alpha2

import (
	cpv1alpha1 "github.com/crossplaneio/crossplane-runtime/apis/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// An OperatingSystem required by a containerised workload.
type OperatingSystem string

// Supported operating system types.
const (
	OperatingSystemLinux   OperatingSystem = "linux"
	OperatingSystemWindows OperatingSystem = "windows"
)

// A CPUArchitecture required by a containerised workload.
type CPUArchitecture string

// Supported architectures
const (
	CPUArchitectureI386  CPUArchitecture = "i386"
	CPUArchitectureAMD64 CPUArchitecture = "amd64"
	CPUArchitectureARM   CPUArchitecture = "arm"
	CPUArchitectureARM64 CPUArchitecture = "arm64"
)

// A ContainerizedWorkloadSpec defines the desired state of a containerized Workload.
type ContainerizedWorkloadSpec struct {
	// OperatingSystem required by this workload.
	// +kubebuilder:validation:Enum=linux;windows
	// +optional
	OperatingSystem *OperatingSystem `json:"osType,omitempty"`

	// CPUArchitecture required by this workload.
	// +kubebuilder:validation:Enum=i386;amd64;arm;arm64
	// +optional
	CPUArchitecture *CPUArchitecture `json:"arch,omitempty"`

	// Containers of which this workload consists.
	Containers []corev1.Container `json:"containers"`
}

// A ResourceReference refers to an resource managed by an OAM resource.
type ResourceReference struct {
	// APIVersion of the referenced resource.
	APIVersion string `json:"apiVersion"`

	// Kind of the referenced resource.
	Kind string `json:"kind"`

	// Name of the referenced resource.
	Name string `json:"name"`

	// UID of the referenced resource.
	// +optional
	UID *types.UID `json:"uid,omitempty"`
}

// A ContainerizedWorkloadStatus represents the observed state of a
// ContainerizedWorkload.
type ContainerizedWorkloadStatus struct {
	cpv1alpha1.ConditionedStatus `json:",inline"`

	// Resources managed by this containerised workload, key the resource UID
	Resources []ResourceReference `json:"resources,omitempty"`
}

// +kubebuilder:object:root=true

// ContainerizedWorkload is the Schema for the containerizedworkloads API
// +kubebuilder:subresource:status
type ContainerizedWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerizedWorkloadSpec   `json:"spec,omitempty"`
	Status ContainerizedWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ContainerizedWorkloadList contains a list of ContainerizedWorkload
type ContainerizedWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerizedWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContainerizedWorkload{}, &ContainerizedWorkloadList{})
}
