/*
Copyright 2021 © MIKAMAI s.r.l

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
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type StatusPhase string

var (
	NoPhase           StatusPhase
	PhaseFailing      StatusPhase = "failing"
	PhaseInitialising StatusPhase = "initialising"
	PhaseActive       StatusPhase = "active"
)

type Connector struct {
	Type    string     `json:"type"`
	Name    string     `json:"name"`
	ID      string     `json:"id"`
	Config  extv1.JSON `json:"config,omitempty"`
	ConfigS string     `json:"-"`
}

// DexSpec defines the desired state of Dex
type DexSpec struct {
	// PublicURL is the publicly reachable URL for the Dex instance, including the path component.
	// Example: https://auth.example.com/dex
	PublicURL string `json:"publicURL"`

	// Connectors is the list of base connectors
	// +kubebuilder:validation:MinItems=1
	Connectors []Connector `json:"connectors"`

	// Replicas is the number of Pods to deploy
	// +kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`

	// EnvFrom is a reference to an environment variables source for the Dex pods
	// +optional
	EnvFrom []v1.EnvFromSource `json:"envFrom,omitempty"`

	// Labels is a set of labels that will be applied to the instance resources
	// +optional
	InstanceLabels map[string]string `json:"instanceLabels,omitempty"`

	// Image is the container image to use. Defaults
	// to the official Dex image and latest tag
	// +optional
	Image string `json:"image,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace
	// to use for pulling Dex images from registries
	// see http://kubernetes.io/docs/user-guide/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run the
	// Dex Pods.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Resources defines resources requests and limits for single Pods.
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector defines which Nodes the Pods are scheduled on.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Affinity defines the pod's scheduling constraints.
	// +optional
	Affinity *v1.Affinity `json:"affinity,omitempty"`

	// Tolerations define the pod's tolerations.
	// +optional
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`

	// TopologySpreadConstraints define the pod's topology spread constraints.
	// +optional
	TopologySpreadConstraints []v1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

	// SecurityContext holds pod-level security attributes and common container settings.
	// This defaults to the default PodSecurityContext.
	// +optional
	SecurityContext *v1.PodSecurityContext `json:"securityContext,omitempty"`

	// Ingress allows to configure the Ingress object to route traffic into Dex
	Ingress Ingress `json:"ingress,omitempty"`
}

type Ingress struct {
	// Annotations to be added to the Ingress object
	Annotations map[string]string `json:"annotations,omitempty"`
	// Labels to be added to the Ingress object
	Labels map[string]string `json:"labels,omitempty"`
	// Enabled allows to turn off the Ingress object (for example for using a LoadBalancer service)
	// +kubebuilder:default:=true
	Enabled *bool `json:"enabled,omitempty"`
	// TLSEnabled toggles the TLS configuration on the Ingress object
	// +optional
	TLSEnabled bool `json:"tlsEnabled,omitempty"`
	// TLSSecretName overrides the generated name for the TLS certificate Secret object
	// +optional
	TLSSecretName string `json:"tlsSecretName,omitempty"`
}

// DexStatus defines the observed state of Dex
type DexStatus struct {
	// Current phase of the operator.
	Phase StatusPhase `json:"phase"`
	// Human-readable message indicating details about current operator phase or error.
	Message string `json:"message"`
	// True if the instance is in a ready state and available for use.
	Ready bool `json:"ready"`
	// Replicas is the current number of replicas
	Replicas int32 `json:"replicas"`
	// Selector is the label selector for the instance pods
	Selector string `json:"selector"`
	// EndpointURL contains the API endpoint for the Dex instance
	EndpointURL string `json:"endpointURL"`
}

type DexConditionType string

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=dexes
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:printcolumn:name="Instances",type=number,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Running",type=number,JSONPath=`.status.replicas`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.publicURL`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,priority=1
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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

func (in *Dex) BuildOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: in.APIVersion,
		Kind:       in.Kind,
		Name:       in.Name,
		UID:        in.UID,
	}
}

func (in *Dex) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace,
	}
}

func (in *Dex) ServiceName() string {
	return fmt.Sprintf("%s-operated", in.Name)
}

func init() {
	SchemeBuilder.Register(&Dex{}, &DexList{})
}
