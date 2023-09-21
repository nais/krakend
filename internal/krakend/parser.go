package krakend

import (
	"encoding/json"
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

func ParseKrakendEndpointsSpec(spec v1.ApiEndpointsSpec) *Partials {
	endpoints := make([]*Endpoint, 0)
	backend := make([]*Backend, 0)

	auth := &AuthValidator{
		OperationDebug: spec.Auth.Debug,
		Alg:            spec.Auth.Alg,
		Cache:          spec.Auth.Cache,
		JwkUrl:         spec.Auth.JwkUrl,
		Issuer:         spec.Auth.Issuer,
		Audience:       spec.Auth.Audience,
		Scope:          spec.Auth.Scope,
	}

	for _, e := range spec.Endpoints {
		backend := append(backend, &Backend{
			Method:     e.Method,
			Host:       []string{e.BackendHost},
			UrlPattern: e.BackendPath,
			Encoding:   DefaultOutputEncoding,
		})
		endpoint := &Endpoint{
			Endpoint:          e.Path,
			Method:            e.Method,
			OutputEncoding:    DefaultOutputEncoding,
			Backend:           backend,
			InputQueryStrings: e.QueryParams,
			InputHeaders:      e.ForwardHeaders,
		}

		extraCfg := &ExtraConfig{}
		if !e.NoAuth {
			extraCfg.AuthValidator = auth
		}
		if e.RateLimit != (v1.RateLimit{}) {
			extraCfg.QosRatelimitRouter = &QosRatelimitRouter{
				MaxRate:        e.RateLimit.MaxRate,
				ClientMaxRate:  e.RateLimit.ClientMaxRate,
				Strategy:       e.RateLimit.Strategy,
				Capacity:       e.RateLimit.Capacity,
				ClientCapacity: e.RateLimit.ClientCapacity,
			}
		}
		if extraCfg.AuthValidator != nil || extraCfg.QosRatelimitRouter != nil {
			endpoint.ExtraConfig = extraCfg
		}
		endpoints = append(endpoints, endpoint)
	}
	return &Partials{
		Endpoints: endpoints,
	}
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

func MergePartials(existing *Partials, n *Partials) (*Partials, error) {
	endpoints := make([]*Endpoint, 0)
	for _, e := range existing.Endpoints {
		endpoints = append(endpoints, e)
	}
	return n, nil
}
