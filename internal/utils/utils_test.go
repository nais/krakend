package utils

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

	up := UniquePaths(endpointsList)
	assert.NoError(t, up)

	err = parseYaml("testdata/apiendpoints_dpaths_diff_app.yaml", endpointsList)
	up = UniquePaths(endpointsList)
	assert.NoError(t, err)
	assert.Error(t, up)

	err = parseYaml("testdata/apiendpoints_dpaths_same_app.yaml", endpointsList)
	up = UniquePaths(endpointsList)
	assert.NoError(t, err)
	assert.Error(t, up)
}

func TestValidateEndpointsList(t *testing.T) {
	apiendpoint := &v1.ApiEndpoints{}
	err := parseYaml("testdata/apiendpoint.yaml", apiendpoint)
	assert.NoError(t, err)

	apiendpointsList := &v1.ApiEndpointsList{}
	err = parseYaml("testdata/apiendpoints.yaml", apiendpointsList)
	assert.NoError(t, err)

	//Validate update of apiendpoint in same apiendpoints resource
	err = ValidateEndpointsList(apiendpointsList, apiendpoint)
	assert.NoError(t, err)

	//Validate update/create of apiendpoint with duplicate path in a different apiendpoints resource
	err = parseYaml("testdata/apiendpoints_in_other_resource.yaml", apiendpointsList)
	assert.NoError(t, err)
	err = ValidateEndpointsList(apiendpointsList, apiendpoint)
	assert.Error(t, err)

	//Validate update/create of apiendpoint with unique paths in different apiendpoints resources
	apiendpoint.Spec.Endpoints[0].Path = "/unique"
	err = ValidateEndpointsList(apiendpointsList, apiendpoint)
	assert.NoError(t, err)

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
