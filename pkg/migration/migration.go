package migration

import (
	"context"
	"fmt"

	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/pkg/migration/kubernetes"
	"github.com/nais/krakend/pkg/migration/parse"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConvertKrakends(ctx context.Context) (string, error) {
	c, ns := kubernetes.NewClient()

	krakends := &krakendv1.KrakendList{}
	err := c.List(ctx, krakends, client.InNamespace(ns))
	if err != nil {
		return "", fmt.Errorf("listing krakends: %v", err)
	}
	apiEndpoints := &krakendv1.ApiEndpointsList{}
	err = c.List(ctx, apiEndpoints, client.InNamespace(ns))
	if err != nil {
		return "", fmt.Errorf("listing apiendpoints: %v", err)
	}

	objs := make([]runtime.Object, 0)
	for _, k := range krakends.Items {
		o, err := parse.Convert(&k, apiEndpoints.Items...)
		if err != nil {
			return "", fmt.Errorf("converting krakend %s: %v", k.Name, err)
		}
		objs = append(objs, o...)
	}
	return parse.ToYAML(objs...)
}
