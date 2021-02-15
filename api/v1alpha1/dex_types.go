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

package v1alpha1

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	DexDefaultVersion = "2.26.0"
)

type Connector struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	ID     string `json:"id"`
	Config string `json:"config,omitempty"`
}

// DexSpec defines the desired state of Dex
type DexSpec struct {
	// PublicHost is the publicly reachable Host for the Dex instance
	PublicHost string `json:"publicHost"`

	// Version is the desired Dex version. Defaults to the latest stable version
	// +optional
	Version string `json:"version,omitempty"`

	// Connectors is the list of base connectors
	// +kubebuilder:validation:MinItems=1
	Connectors []Connector `json:"connectors"`

	// Replicas is the number of pods to deploy
	// +kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`

	// EnvFrom is a reference to an environment variables source for the Dex pods
	// +optional
	EnvFrom []v1.EnvFromSource `json:"envFrom,omitempty"`

	// Labels is a set of labels that will be applied to the instance resources
	InstanceLabels map[string]string `json:"instanceLabels"`
}

// DexStatus defines the observed state of Dex
type DexStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The generation observed by the Dex controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Represents the latest available observations of a deployment's current state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []DexCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,6,rep,name=conditions"`
}

type DexConditionType string

// These are valid conditions of a deployment.
const (
	DexReady       DexConditionType = "Ready"
	DexProgressing DexConditionType = "Progressing"
	DexFailure     DexConditionType = "Failure"
)

// DeploymentCondition describes the state of a deployment at a certain point.
type DexCondition struct {
	// Type of deployment condition.
	Type DexConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=dexes
// +kubebuilder:subresource:status

// Dex is the Schema for the dexes API
type Dex struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexSpec   `json:"spec,omitempty"`
	Status DexStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexList contains a list of Dex
type DexList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dex `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dex{}, &DexList{})
}

func (in *Dex) BuildOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: in.APIVersion,
		Kind:       in.Kind,
		Name:       in.Name,
		UID:        in.UID,
	}
}

func (in *Dex) Version() string {
	if in.Spec.Version == "" {
		in.Spec.Version = DexDefaultVersion
	}

	return strings.TrimPrefix(in.Spec.Version, "v")
}

func (in *Dex) ServiceName() string {
	return fmt.Sprintf("%s-operated", in.Name)
}
