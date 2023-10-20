package krakend

import (
	"encoding/json"
	"fmt"
	v1 "github.com/nais/krakend/api/v1"
)

type Partials struct {
	Endpoints []*Endpoint `json:"endpoints,omitempty"`
}
type Endpoint struct {
	Endpoint          string       `json:"endpoint"`
	Method            string       `json:"method"`
	OutputEncoding    string       `json:"output_encoding"`
	InputQueryStrings []string     `json:"input_query_strings,omitempty"`
	InputHeaders      []string     `json:"input_headers,omitempty"`
	ExtraConfig       *ExtraConfig `json:"extra_config,omitempty"`
	Backend           []*Backend   `json:"backend"`
}

type Backend struct {
	Method     string   `json:"method"`
	Host       []string `json:"host"`
	UrlPattern string   `json:"url_pattern"`
	Encoding   string   `json:"encoding"`
}

type ExtraConfig struct {
	AuthValidator      *AuthValidator      `json:"auth/validator,omitempty"`
	QosRatelimitRouter *QosRatelimitRouter `json:"qos/ratelimit/router,omitempty"`
}

type AuthValidator struct {
	OperationDebug bool     `json:"operation_debug"`
	Alg            string   `json:"alg"`
	Cache          bool     `json:"cache"`
	JwkUrl         string   `json:"jwk_url"`
	Issuer         string   `json:"issuer"`
	Audience       []string `json:"audience,omitempty"`
	Scope          []string `json:"scope,omitempty"`
}

type QosRatelimitRouter struct {
	MaxRate        int    `json:"max_rate"`
	ClientMaxRate  int    `json:"client_max_rate"`
	Strategy       string `json:"strategy"`
	Capacity       int    `json:"capacity"`
	ClientCapacity int    `json:"client_capacity"`
}

const DefaultOutputEncoding = "no-op"

func ToKrakendEndpoints(k *v1.Krakend, list []v1.ApiEndpoints) ([]*Endpoint, error) {
	endpoints := make([]*Endpoint, 0)
	for _, item := range list {
		parsed, err := parseKrakendEndpointsSpec(k, item.Spec)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, parsed...)
	}
	return endpoints, nil
}

func parseKrakendEndpointsSpec(k *v1.Krakend, spec v1.ApiEndpointsSpec) ([]*Endpoint, error) {
	endpoints := make([]*Endpoint, 0)

	auth, err := findAuthProvider(k, &spec.Auth)
	rateLimit := spec.RateLimit
	if err != nil {
		return nil, err
	}

	for _, e := range spec.Endpoints {
		endpoint := parseEndpoint(e)
		endpoint.ExtraConfig.AuthValidator = auth
		endpoint.ExtraConfig.QosRatelimitRouter = parseRateLimit(rateLimit)
		endpoints = append(endpoints, endpoint)
	}
	for _, e := range spec.OpenEndpoints {
		endpoint := parseEndpoint(e)
		endpoint.ExtraConfig = &ExtraConfig{}
		endpoint.ExtraConfig.QosRatelimitRouter = parseRateLimit(rateLimit)
		endpoints = append(endpoints, endpoint)
	}
	return endpoints, nil
}

func parseEndpoint(e v1.Endpoint) *Endpoint {
	backend := []*Backend{
		{
			Method:     e.Method,
			Host:       []string{e.BackendHost},
			UrlPattern: e.BackendPath,
			Encoding:   DefaultOutputEncoding,
		},
	}
	endpoint := &Endpoint{
		Endpoint:          e.Path,
		Method:            e.Method,
		OutputEncoding:    DefaultOutputEncoding,
		Backend:           backend,
		InputQueryStrings: e.QueryParams,
		InputHeaders:      e.ForwardHeaders,
	}

	extraCfg := &ExtraConfig{}
	endpoint.ExtraConfig = extraCfg
	return endpoint
}

func parseRateLimit(r *v1.RateLimit) *QosRatelimitRouter {
	if r == nil {
		return nil
	}
	return &QosRatelimitRouter{
		MaxRate:        r.MaxRate,
		ClientMaxRate:  r.ClientMaxRate,
		Strategy:       r.Strategy,
		Capacity:       r.Capacity,
		ClientCapacity: r.ClientCapacity,
	}
}

func findAuthProvider(k *v1.Krakend, auth *v1.Auth) (*AuthValidator, error) {
	for _, p := range k.Spec.AuthProviders {
		if p.Name == auth.Name {
			return &AuthValidator{
				OperationDebug: auth.Debug,
				Alg:            p.Alg,
				Cache:          auth.Cache,
				JwkUrl:         p.JwkUrl,
				Issuer:         p.Issuer,
				Audience:       auth.Audience,
				Scope:          auth.Scope,
			}, nil
		}
	}
	return nil, fmt.Errorf("auth provider with name '%s' not found", auth.Name)
}

func ParsePartials(content []byte) (*Partials, error) {
	partials := &Partials{}
	endpoints := make([]*Endpoint, 0)
	err := json.Unmarshal(content, &endpoints)
	if err != nil {
		return nil, err
	}
	partials.Endpoints = endpoints
	return partials, nil
}
