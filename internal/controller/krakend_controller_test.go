package controller

import (
	"fmt"
	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/helm"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chartutil"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"testing"
)

func TestRenderChart(t *testing.T) {
	k, err := unmarshallKrakend("testdata/krakend_min.yaml")
	assert.NoError(t, err)

	values, err := prepareValues(k)
	assert.NoError(t, err)

	c, err := helm.LoadChart("testdata/krakend")
	assert.NoError(t, err)

	resources, err := c.ToUnstructured(k.Spec.Name, chartutil.Values{
		"krakend": values,
	})
	assert.NoError(t, err)

	for _, r := range resources {
		fmt.Printf("%+v\n", r)
		if r.GetKind() == "Deployment" {
			assert.Equal(t, 2, r.Object["spec"].(map[string]interface{})["replicas"])
			return
		}
	}
}

func unmarshallKrakend(yamlFile string) (*krakendv1.Krakend, error) {
	sch := runtime.NewScheme()
	_ = scheme.AddToScheme(sch)
	err := krakendv1.AddToScheme(sch)
	if err != nil {
		return nil, err
	}
	_ = apiextv1beta1.AddToScheme(sch)
	decode := serializer.NewCodecFactory(sch).UniversalDeserializer().Decode
	stream, _ := os.ReadFile(yamlFile)
	obj, gvk, err := decode(stream, nil, nil)
	if err != nil {
		return nil, err
	}

	if gvk.Kind == "Krakend" {
		return obj.(*krakendv1.Krakend), nil
	}
	return nil, fmt.Errorf("kind is not krakend")
}
