package controller

import (
	v1 "github.com/nais/krakend/api/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"testing"
)

func TestUniquePaths(t *testing.T) {
	endpointsList := &v1.ApiEndpointsList{}
	err := parseYaml("testdata/apiendpoints.yaml", endpointsList)
	assert.NoError(t, err)

	up := uniquePaths(endpointsList)
	assert.True(t, up)

	err = parseYaml("testdata/apiendpoints_dpaths_diff_app.yaml", endpointsList)
	up = uniquePaths(endpointsList)
	assert.NoError(t, err)
	assert.False(t, up)

	err = parseYaml("testdata/apiendpoints_dpaths_same_app.yaml", endpointsList)
	up = uniquePaths(endpointsList)
	assert.NoError(t, err)
	assert.False(t, up)
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