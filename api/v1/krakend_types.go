package v1

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// KrakendSpec defines the desired state of Krakend
type KrakendSpec struct {
	// Ingress lets you configure the ingress class, annotations and hosts or tls for an ingress
	Ingress Ingress `json:"ingress,omitempty"`
	// IngressHost is a shortcut for creating a single host ingress with sane defaults, if Ingress is specified this is ignored
	IngressHost string `json:"ingressHost,omitempty"`
	// AuthProviders is a list of supported auth providers to be used in ApiEndpoints
	AuthProviders []AuthProvider `json:"authProviders,omitempty" fakesize:"1"`
	// Deployment defines configuration for the KrakenD deployment
	Deployment KrakendDeployment `json:"deployment,omitempty"`
}

// AuthProvider defines the configuration for an JWT auth provider
type AuthProvider struct {
	// Name is the name of the auth provider, e.g. maskinporten
	Name string `json:"name"`
	// Alg is the algorithm used for signing the JWT token, e.g. RS256
	Alg string `json:"alg"`
	// JwkUrl is the URL to the JWKs for the auth provider
	JwkUrl string `json:"jwkUrl"`
	// Issuer is the issuer of the JWT token
	Issuer string `json:"issuer"`
}

// KrakendDeployment defines the configuration for the KrakenD deployment
type KrakendDeployment struct {
	// DeploymentType is the type of deployment to use, either deployment or rollout
	DeploymentType string `json:"deploymentType,omitempty"`
	// ReplicaCount is the number of replicas to use for the deployment
	ReplicaCount int `json:"replicaCount,omitempty"`
	// Resources is the resource requirements for the deployment, as in corev1.ResourceRequirements
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Image is the image configuration to use for the deployment
	Image Image `json:"image,omitempty"`
	// ExtraEnvVars is a list of extra environment variables to add to the deployment
	ExtraEnvVars []corev1.EnvVar `json:"extraEnvVars,omitempty"`
	// ExtraConfig is an object, defining extra config variables to use for the deployment
	ExtraConfig *apiextensionsv1.JSON `json:"extraConfig,omitempty"`
}

// Ingress defines the ingress configuration
type Ingress struct {
	// Enabled is whether to enable ingress for the Krakend instance
	Enabled bool `json:"enabled,omitempty"`
	// Class is the ingress class to use for the Krakend instance
	ClassName string `json:"className,omitempty"`
	// Annotations is a list of annotations to add to the ingress
	Annotations map[string]string `json:"annotations,omitempty"`
	// Hosts is a list of hosts to add to the ingress
	Hosts []Host `json:"hosts,omitempty"`
}

// Host defines the host configuration for an ingress
type Host struct {
	// Host is the host name to add to the ingress
	Host string `json:"host,omitempty"`
	// Paths is a list of paths to add to the ingress
	Paths []Path `json:"paths,omitempty"`
}

// Path defines the path configuration for an ingress
type Path struct {
	// Path is the path to add to the ingress
	Path string `json:"path,omitempty"`
	// PathType is the path type to add to the ingress
	PathType string `json:"pathType,omitempty"`
}

// Image defines the image configuration for the Krakend deployment
type Image struct {
	// Registry is the registry to use for the image
	Registry string `json:"registry,omitempty"`
	// Repository is the repository to use for the image
	Repository string `json:"repository,omitempty"`
	// Tag is the tag to use for the image
	Tag string `json:"tag,omitempty"`
	// PullPolicy is the pull policy to use for the image
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
