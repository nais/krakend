package krakend

import (
	"encoding/json"
	v1 "github.com/nais/krakend/api/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"testing"
)

// TODO: add testcases
func TestParseKrakendEndpointsSpec(t *testing.T) {
	endpoints := &v1.ApiEndpoints{}
	err := parseYaml("testdata/apiendpoints.yaml", endpoints)
	assert.NoError(t, err)

	k := &v1.Krakend{}
	err = parseYaml("testdata/krakend.yaml", k)
	assert.NoError(t, err)

	partials, err := parseKrakendEndpointsSpec(k, endpoints.Spec)
	assert.NoError(t, err)

	_, err = json.Marshal(partials)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(partials))
	p := partials[0]
	assert.Equal(t, "/echo", p.Endpoint)
	assert.Equal(t, "GET", p.Method)
	assert.Equal(t, "/", p.Backend[0].UrlPattern)
	assert.Equal(t, "GET", p.Backend[0].Method)
	assert.Equal(t, "http://echo:1027", p.Backend[0].Host[0])
	assert.Equal(t, "org1:team1:krakend.app", p.ExtraConfig.AuthValidator.Scope[0])
	assert.Equal(t, "https://test.maskinporten.no/jwk", p.ExtraConfig.AuthValidator.JwkUrl)
	assert.Equal(t, "https://test.maskinporten.no/", p.ExtraConfig.AuthValidator.Issuer)
	assert.Equal(t, "RS256", p.ExtraConfig.AuthValidator.Alg)
	assert.Equal(t, 10, p.ExtraConfig.QosRatelimitRouter.MaxRate)
	assert.Equal(t, 0, p.ExtraConfig.QosRatelimitRouter.ClientMaxRate)
	assert.Equal(t, "ip", p.ExtraConfig.QosRatelimitRouter.Strategy)
	assert.Equal(t, 0, p.ExtraConfig.QosRatelimitRouter.Capacity)
	assert.Equal(t, 0, p.ExtraConfig.QosRatelimitRouter.ClientCapacity)
	assert.Equal(t, "foo", p.InputQueryStrings[0])
	assert.Equal(t, "bar", p.InputQueryStrings[1])

	p2 := partials[1]
	assert.Equal(t, "/doc", p2.Endpoint)
	assert.Equal(t, "GET", p2.Method)
	assert.Equal(t, "/doc", p2.Backend[0].UrlPattern)
	assert.Equal(t, "GET", p2.Backend[0].Method)
	assert.Equal(t, "http://echo:1027", p2.Backend[0].Host[0])
	assert.Empty(t, p2.ExtraConfig.AuthValidator)
	assert.Empty(t, p2.ExtraConfig.QosRatelimitRouter)
}

func parseYaml(file string, v any) error {
	reader, err := os.Open(file)
	if err != nil {
		return err
	}
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	err = decoder.Decode(v)
	if err != nil {
		return err
	}
	return nil
}

func TestParsePartials(t *testing.T) {
	content, err := os.ReadFile("testdata/config.json")
	assert.NoError(t, err)

	partials, err := ParsePartials(content)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(partials.Endpoints))
}
