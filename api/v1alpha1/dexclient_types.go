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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DexClientSpec defines the desired state of DexClient
type DexClientSpec struct {
	// Id is the Dex clientId
	Id string `json:"id"`

	// Name is the Dex client name
	Name string `json:"name"`

	// RedirectUris is the list of callback URIs for the client
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Format=uri
	RedirectUris []string `json:"redirectUris"`

	// Public marks the client as a public OAuth client
	// +kubebuilder:default:=false
	Public bool `json:"public,omitempty"`

	// InstanceSelector is used to select the target Dex instance
	InstanceSelector *metav1.LabelSelector `json:"instanceSelector"`
}

// DexClientStatus defines the observed state of DexClient
type DexClientStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// DexClient is the Schema for the dexclients API
type DexClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexClientSpec   `json:"spec,omitempty"`
	Status DexClientStatus `json:"status,omitempty"`
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
