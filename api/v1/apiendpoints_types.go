package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Endpoint defines the endpoint configuration
type Endpoint struct {
	// Path is exact path of an endpoint in a KrakenD instance and must be unique, see https://www.krakend.io/docs/endpoints/#endpoint
	Path string `json:"path,omitempty" fake:"{inputname}"`
	// Method is the HTTP method of the endpoint, see https://www.krakend.io/docs/endpoints/#method
	Method string `json:"method,omitempty" fake:"GET"`
	// BackendHost is the base URL of the backend service and must start with the protocol, i.e. http:// or https://
	BackendHost string `json:"backendHost,omitempty" fake:"http://appname.namespace.svc.cluster.local"`
	// BackendPath is the path of the backend service and follows the conventions of url_pattern in https://www.krakend.io/docs/backends/#backendupstream-configuration
	BackendPath string `json:"backendPath,omitempty" fake:"{inputname}"`
	// ForwardHeaders is a list of header names to be forwarded to the backend service, see https://www.krakend.io/docs/endpoints/#input_headers
	ForwardHeaders []string `json:"forwardHeaders,omitempty" fake:"{word}" fakesize:"1"`
	// QueryParams is an exact list of query parameter names that are allowed to reach the backend. By default, KrakenD won’t pass any query string to the backend, see https://www.krakend.io/docs/endpoints/#input_query_strings
	QueryParams []string `json:"queryParams,omitempty" fake:"{word}" fakesize:"1"`
}

// RateLimit defines the rate limit configuration
type RateLimit struct {
	// MaxRate is documented here: https://www.krakend.io/docs/endpoints/rate-limit/#configuration
	MaxRate int `json:"maxRate,omitempty" fake:"5"`
	// ClientMaxRate is documented here: https://www.krakend.io/docs/endpoints/rate-limit/#configuration
	ClientMaxRate int `json:"clientMaxRate,omitempty" fake:"{number:10,100}"`
	// Every is documented here: https://www.krakend.io/docs/endpoints/rate-limit/#configuration
	Every string `json:"every,omitempty" fake:"10s"`
	// Key is documented here: https://www.krakend.io/docs/endpoints/rate-limit/#configuration
	Key string `json:"key,omitempty" fake:"X-TOKEN"`
	// Strategy is documented here: https://www.krakend.io/docs/endpoints/rate-limit/#configuration
	Strategy string `json:"strategy,omitempty" fake:"ip"`
	// Capacity is documented here: https://www.krakend.io/docs/endpoints/rate-limit/#configuration
	Capacity int `json:"capacity,omitempty" fake:"1000"`
	// ClientCapacity is documented here: https://www.krakend.io/docs/endpoints/rate-limit/#configuration
	ClientCapacity int `json:"clientCapacity,omitempty" fake:"{number:10,100}"`
}

// Auth defines the JWT authentication config
type Auth struct {
	// Name is the name of the auth provider defined in the Krakend resource, e.g. maskinporten
	Name string `json:"name" fake:"maskinporten"`
	// Cache is whether to cache the JWKs from the auth provider
	Cache bool `json:"cache,omitempty" fake:"true"`
	// Debug is whether to enable debug logging for the auth provider
	Debug bool `json:"debug,omitempty" fake:"false"`
	// Audience is the list of audiences to validate the JWT against
	Audience []string `json:"audience,omitempty" fake:"{uuid}" fakesize:"1"`
	// Scope is the list of scopes to validate the JWT against
	Scope []string `json:"scope,omitempty" fake:"{word}" fakesize:"1"`
}

// ApiEndpointsSpec defines the desired state of ApiEndpoints
type ApiEndpointsSpec struct {
	// Krakend is the name of the Krakend instance in the cluster
	Krakend string `json:"krakend,omitempty" fake:"skip"`
	// AppName is the name of the API, e.g. name of the application or service
	AppName string `json:"appName,omitempty" fake:"{appname}"`
	// Auth is the common JWT authentication provider used for the endpoints specified in Endpoints
	Auth Auth `json:"auth,omitempty"`
	// RateLimit is the common rate limit configuration used for the endpoints specified in Endpoints and OpenEndpoints
	RateLimit *RateLimit `json:"rateLimit,omitempty"`
	// Endpoints is a list of endpoints that require authentication
	Endpoints []Endpoint `json:"endpoints,omitempty" fakesize:"1"`
	// OpenEndpoints is a list of endpoints that do not require authentication
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
