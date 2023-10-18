package helm

import (
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

const krakendChart = "../../charts/krakend"

func TestChart_ToUnstructured(t *testing.T) {
	c, err := LoadChart(krakendChart)
	assert.NoError(t, err)
	rs, err := c.ToUnstructured("my-release", chartutil.Values{
		"krakend": map[string]interface{}{
			"replicaCount": 3,
		},
	})
	assert.NoError(t, err)
	for _, r := range rs {
		if r.GetKind() == "Deployment" {
			d := &v1.Deployment{}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(r.Object, d)
			assert.NoError(t, err)
			replicas := *d.Spec.Replicas
			assert.Equal(t, int32(3), replicas)
			assert.True(t, len(d.Spec.Template.Spec.Containers) == 1)
			assert.True(t, len(d.Spec.Template.Spec.Containers[0].Env) > 0)
			found := false
			for _, e := range d.Spec.Template.Spec.Containers[0].Env {
				if e.Name == "USAGE_DISABLE" {
					found = true
					assert.Equalf(t, "1", e.Value, "env var USAGE_DISABLE should have 1 as value")
					break
				}
			}
			assert.Truef(t, found, "env var USAGE_DISABLE not found in deployment")
			return
		}
	}

	assert.Fail(t, "deployment not found")
}
