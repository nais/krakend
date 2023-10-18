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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// KrakendSpec defines the desired state of Krakend
type KrakendSpec struct {
	Name string `json:"name"`
	// Ingress lets you configure the ingress class, annotations and hosts or tls for an ingress
	Ingress Ingress `json:"ingress,omitempty"`
	// IngressHost is a shortcut for creating a single host ingress with sane defaults, if Ingress is specified this is ignored
	IngressHost   string            `json:"ingressHost,omitempty"`
	AuthProviders []AuthProvider    `json:"authProviders,omitempty"`
	Deployment    KrakendDeployment `json:"deployment,omitempty"`
}

type AuthProvider struct {
	Name   string `json:"name"`
	Alg    string `json:"alg"`
	JwkUrl string `json:"jwkUrl"`
	Issuer string `json:"issuer"`
}

type KrakendDeployment struct {
	DeploymentType string                      `json:"deploymentType,omitempty"`
	ReplicaCount   int                         `json:"replicaCount,omitempty"`
	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	Image          Image                       `json:"image,omitempty"`
	// ExtraEnv is a list of extra environment variables to add to the deployment
	// +kubebuilder:validation:Optional
	ExtraEnvVars []corev1.EnvVar `json:"extraEnvVars,omitempty"`
}

type Ingress struct {
	Enabled     bool              `json:"enabled,omitempty"`
	ClassName   string            `json:"className,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Hosts       []Host            `json:"hosts,omitempty"`
}

type Host struct {
	Host  string `json:"host,omitempty"`
	Paths []Path `json:"paths,omitempty"`
}

type Path struct {
	Path     string `json:"path,omitempty"`
	PathType string `json:"pathType,omitempty"`
}

type Image struct {
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	PullPolicy string `json:"pullPolicy,omitempty"`
}

// KrakendStatus defines the observed state of Krakend
type KrakendStatus struct {
	SynchronizationTimestamp metav1.Time `json:"synchronizationTimestamp,omitempty"`
	SynchronizationHash      string      `json:"synchronizationHash,omitempty"`
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

func (k *Krakend) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: k.Namespace,
		Name:      k.Name,
	}
}
