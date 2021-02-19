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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// DexClientSpec defines the desired state of DexClient
type DexClientSpec struct {
	// Name is the Dex client name
	Name string `json:"name"`

	// RedirectUris is the list of callback URIs for the client
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Format=uri
	RedirectUris []string `json:"redirectUris"`

	// Public marks the client as a public OAuth client
	// +kubebuilder:default:=false
	Public bool `json:"public,omitempty"`

	// InstanceRef is used to select the target Dex instance
	// Cannot be updated
	InstanceRef InstanceRef `json:"instanceRef"`

	// ClientIDKey allows to override the key used in the generated Secret for the clientID
	// +kubebuilder:default:=clientID
	// +optional
	ClientIDKey string `json:"clientIDKey"`

	// ClientSecretKey allows to override the key used in the generated Secret for the clientSecret
	// +kubebuilder:default:=clientSecret
	// +optional
	ClientSecretKey string `json:"clientSecretKey"`
}

type InstanceRef struct {
	// Name is the object name for the Dex instance
	// Cannot be updated
	Name string `json:"name"`
	// Namespace is the object name for the Dex instance
	// Cannot be updated
	// If empty will default to the same namespace as the DexClient
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// DexClientStatus defines the observed state of DexClient
type DexClientStatus struct {
	// Phase is the current phase of the operator.
	Phase StatusPhase `json:"phase"`
	// Message is a human-readable message indicating details about current operator phase or error.
	Message string `json:"message"`
	// Ready will be true if the client is in a ready state and available for use.
	Ready bool `json:"ready"`
	// ClientID is the generated OAuth client_id for this client
	ClientID string `json:"clientID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=dexclients
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Client ID",type=string,JSONPath=`.status.clientID`
// +kubebuilder:printcolumn:name="Public",type=boolean,JSONPath=`.spec.public`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexClient is the Schema for the dexclients API
type DexClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexClientSpec   `json:"spec,omitempty"`
	Status DexClientStatus `json:"status,omitempty"`
}

func (in *DexClient) ClientID() string {
	return fmt.Sprintf("%s-%s", in.Namespace, in.Name)
}

// +kubebuilder:object:root=true

// DexClientList contains a list of DexClient
type DexClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexClient{}, &DexClientList{})
}

func (in *DexClient) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      in.Name,
		Namespace: in.Namespace,
	}
}
