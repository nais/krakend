package helm

import (
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"testing"
)

func TestChart_ToUnstructured(t *testing.T) {
	c, err := LoadChart("testdata/krakend-v0.1.21.tgz")
	assert.NoError(t, err)
	rs, err := c.ToUnstructured("my-release", chartutil.Values{
		"replicaCount": 2,
	})
	assert.NoError(t, err)
	for _, r := range rs {
		if r.GetKind() == "Deployment" {
			assert.Equal(t, 2, r.Object["spec"].(map[string]interface{})["replicas"])
			return
		}
	}
	assert.Fail(t, "deployment not found")
}

func TestChart_Render(t *testing.T) {
	c, err := LoadChart("testdata/krakend-v0.1.21.tgz")
	if err != nil {
		t.Fatal(err)
	}
	files, err := c.render("my-release", chartutil.Values{
		"replicaCount": 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(files["krakend/templates/deployment.yaml"]), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	deployment := obj.(*v1.Deployment)

	assert.Equal(t, 2, int(*deployment.Spec.Replicas))
	assert.Equal(t, "my-release-krakend", deployment.Name)
}
