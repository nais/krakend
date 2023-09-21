package krakend

import (
	"encoding/json"
	"fmt"
	v1 "github.com/nais/krakend/api/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"testing"
)

// TODO: add testcases
func TestParseKrakendEndpointsSpec(t *testing.T) {
	reader, err := os.Open("testdata/krakend-endpoints.yaml")
	assert.NoError(t, err)
	endpoints := &v1.ApiEndpoints{}
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	err = decoder.Decode(endpoints)
	assert.NoError(t, err)
	fmt.Printf("%+v\n", endpoints)

	partials := ParseKrakendEndpointsSpec(endpoints.Spec)

	out, err := json.Marshal(partials.Endpoints)
	assert.NoError(t, err)
	os.WriteFile("testdata/krakend-endpoints-parsed.json", out, 0644)
}

func TestParsePartials(t *testing.T) {
	content, err := os.ReadFile("testdata/config.json")
	assert.NoError(t, err)

	partials, err := ParsePartials(content)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(partials.Endpoints))
}
