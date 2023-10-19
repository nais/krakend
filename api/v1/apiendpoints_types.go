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
	Path           string     `json:"path,omitempty" fake:"{inputname}"`
	Method         string     `json:"method,omitempty" fake:"GET"`
	BackendHost    string     `json:"backendHost,omitempty" fake:"http://appname.namespace.svc.cluster.local"`
	BackendPath    string     `json:"backendPath,omitempty" fake:"{inputname}"`
	ForwardHeaders []string   `json:"forwardHeaders,omitempty" fake:"{word}" fakesize:"1"`
	QueryParams    []string   `json:"queryParams,omitempty" fake:"{word}" fakesize:"1"`
	RateLimit      *RateLimit `json:"rateLimit,omitempty"`
}

type RateLimit struct {
	MaxRate        int    `json:"maxRate" fake:"5"`
	ClientMaxRate  int    `json:"clientMaxRate" fake:"{number:10,100}"`
	Strategy       string `json:"strategy" fake:"ip"`
	Capacity       int    `json:"capacity" fake:"1000"`
	ClientCapacity int    `json:"clientCapacity" fake:"{number:10,100}"`
}

type Auth struct {
	// Name is the name of the auth provider defined in the Krakend resource, e.g. maskinporten
	Name     string   `json:"name" fake:"maskinporten"`
	Cache    bool     `json:"cache,omitempty" fake:"true"`
	Debug    bool     `json:"debug,omitempty" fake:"false"`
	Audience []string `json:"audience,omitempty" fake:"{uuid}" fakesize:"1"`
	Scope    []string `json:"scope,omitempty" fake:"{word}" fakesize:"1"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApiEndpointsSpec defines the desired state of ApiEndpoints
type ApiEndpointsSpec struct {
	// Krakend is the name of the Krakend instance in the cluster
	Krakend string `json:"krakend,omitempty" fake:"skip"`
	// AppName is the name of the API, e.g. name of the application or service
	AppName       string     `json:"appName,omitempty" fake:"{appname}"`
	Auth          Auth       `json:"auth,omitempty"`
	Endpoints     []Endpoint `json:"endpoints,omitempty" fakesize:"1"`
	OpenEndpoints []Endpoint `json:"openEndpoints,omitempty" fakesize:"1"`
}

// ApiEndpointsStatus defines the observed state of ApiEndpoints
type ApiEndpointsStatus struct {
	SynchronizationTimestamp metav1.Time `json:"synchronizationTimestamp,omitempty"`
	SynchronizationHash      string      `json:"synchronizationHash,omitempty"`
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
