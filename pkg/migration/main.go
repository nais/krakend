package main

import (
	"context"
	"time"

	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/pkg/migration/kubernetes"
	"github.com/nais/krakend/pkg/migration/parse"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, ns := kubernetes.NewClient()

	krakends := &krakendv1.KrakendList{}
	err := c.List(ctx, krakends, client.InNamespace(ns))
	if err != nil {
		log.Fatalf("listing krakends: %v", err)
	}
	apiEndpoints := &krakendv1.ApiEndpointsList{}
	err = c.List(ctx, apiEndpoints, client.InNamespace(ns))
	if err != nil {
		log.Fatalf("listing apiendpoints: %v", err)
	}

	objs := make([]runtime.Object, 0)
	for _, k := range krakends.Items {
		o, err := parse.Convert(&k, apiEndpoints.Items...)
		if err != nil {
			log.Fatalf("converting krakend %s: %v", k.Name, err)
		}
		objs = append(objs, o...)
	}
	out := OutputYaml(objs...)
	println(out)
}

func OutputYaml(v ...runtime.Object) string {
	s, err := parse.ToYAML(v...)
	if err != nil {
		log.Fatalf("marshalling object: %v", err)
	}
	return s
}
