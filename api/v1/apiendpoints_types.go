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

type Endpoint struct {
	Path           string    `json:"path,omitempty"`
	Method         string    `json:"method,omitempty"`
	BackendHost    string    `json:"backendHost,omitempty"`
	BackendPath    string    `json:"backendPath,omitempty"`
	ForwardHeaders []string  `json:"forwardHeaders,omitempty"`
	QueryParams    []string  `json:"queryParams,omitempty"`
	NoAuth         bool      `json:"noAuth,omitempty"`
	RateLimit      RateLimit `json:"rateLimit,omitempty"`
}

type RateLimit struct {
	MaxRate        int    `json:"maxRate"`
	ClientMaxRate  int    `json:"clientMaxRate"`
	Strategy       string `json:"strategy"`
	Capacity       int    `json:"capacity"`
	ClientCapacity int    `json:"clientCapacity"`
}

type Auth struct {
	Alg      string   `json:"alg"`
	Cache    bool     `json:"cache,omitempty"`
	Debug    bool     `json:"debug,omitempty"`
	JwkUrl   string   `json:"jwkUrl"`
	Issuer   string   `json:"issuer"`
	Audience []string `json:"audience,omitempty"`
	Scope    []string `json:"scope,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApiEndpointsSpec defines the desired state of ApiEndpoints
type ApiEndpointsSpec struct {
	// KrakendInstance is the name of the Krakend instance in the cluster
	KrakendInstance string `json:"krakendInstance"`
	// ApiName is the name of the API, e.g. name of the application or service
	ApiName   string     `json:"apiName,omitempty"`
	Auth      Auth       `json:"auth,omitempty"`
	Endpoints []Endpoint `json:"endpoints,omitempty"`
}

// ApiEndpointsStatus defines the observed state of ApiEndpoints
type ApiEndpointsStatus struct {
	// TODO: add status fields here
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ApiEndpoints is the Schema for the apiendpoints API
type ApiEndpoints struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApiEndpointsSpec   `json:"spec,omitempty"`
	Status ApiEndpointsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ApiEndpointsList contains a list of ApiEndpoints
type ApiEndpointsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApiEndpoints `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApiEndpoints{}, &ApiEndpointsList{})
}
