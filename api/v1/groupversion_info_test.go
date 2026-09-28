package v1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestAddToSchemeRegistersAPIResources(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("add API resources to scheme: %v", err)
	}

	for _, kind := range []string{
		"Krakend",
		"KrakendList",
		"ApiEndpoints",
		"ApiEndpointsList",
	} {
		gvk := GroupVersion.WithKind(kind)
		if _, err := scheme.New(gvk); err != nil {
			t.Errorf("create %s from scheme: %v", gvk, err)
		}
	}
}
