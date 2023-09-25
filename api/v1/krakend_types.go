/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PartialsConfigMap struct {
	Name         string `json:"name"`
	EndpointsKey string `json:"endpointsKey"`
}

type ConfigConfigMap struct {
	Name string `json:"name"`
}

// KrakendSpec defines the desired state of Krakend
type KrakendSpec struct {
	Name              string            `json:"name"`
	Ingress           string            `json:"ingress"`
	PartialsConfigMap PartialsConfigMap `json:"partialsConfigMap"`
	ConfigConfigMap   ConfigConfigMap   `json:"configConfigMap"`
}

// KrakendStatus defines the observed state of Krakend
type KrakendStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Krakend is the Schema for the krakends API
type Krakend struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrakendSpec   `json:"spec,omitempty"`
	Status KrakendStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KrakendList contains a list of Krakend
type KrakendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Krakend `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Krakend{}, &KrakendList{})
}
