package krakend

import (
	"encoding/json"
	"os"
	"testing"

	v1 "github.com/nais/krakend/api/v1"
	"github.com/stretchr/testify/assert"
)

const goldenConfig = "testdata/krakend.json"

// Run with UPDATE_GOLDEN=1 to regenerate testdata/krakend.json
func TestBuildConfig(t *testing.T) {
	k := &v1.Krakend{}
	assert.NoError(t, parseYaml("testdata/krakend.yaml", k))
	e := v1.ApiEndpoints{}
	assert.NoError(t, parseYaml("testdata/apiendpoints.yaml", &e))

	other := *e.DeepCopy()
	other.Name = "aaa-endpoints"
	other.Spec.Endpoints[0].Path = "/other"
	other.Spec.OpenEndpoints = nil

	actual, err := BuildConfig(k, []v1.ApiEndpoints{e, other})
	assert.NoError(t, err)

	cfg := &Config{}
	assert.NoError(t, json.Unmarshal(actual, cfg))
	assert.Equal(t, 4, cfg.Version)
	assert.Len(t, cfg.Endpoints, 3)
	assert.Equal(t, "/other", cfg.Endpoints[0].Endpoint, "endpoints should be sorted by ApiEndpoints name")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		assert.NoError(t, os.WriteFile(goldenConfig, actual, 0o600))
	}
	expected, err := os.ReadFile(goldenConfig)
	assert.NoError(t, err)
	assert.Equal(t, string(expected), string(actual))
}

func TestBuildConfigNoEndpoints(t *testing.T) {
	actual, err := BuildConfig(&v1.Krakend{}, nil)
	assert.NoError(t, err)
	assert.Contains(t, string(actual), `"endpoints": []`)
}
