package parse

import (
	"encoding/json"
	"fmt"

	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/krakend"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConfigMapEndpointsKey = "endpoints.json"
)

func ToKrakendConfig(k *krakendv1.Krakend) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := ParseYaml(templatesDir, "templates/krakend_config.yaml", &cm)
	if err != nil {
		return cm, fmt.Errorf("parsing krakend config template: %v", err)
	}
	cm.Name = resourceName(k, "config")
	cm.Namespace = k.Namespace
	return cm, nil
}

func ToPartialsConfig(k *krakendv1.Krakend, list []krakendv1.ApiEndpoints) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
	}
	cmName := resourceName(k, "partials")
	cm.Namespace = k.Namespace
	cm.Name = cmName
	cm.Data = map[string]string{}
	cm.Annotations = map[string]string{}
	cm.Annotations["reloader.stakater.com/match"] = "true"

	allEndpoints, err := krakend.ToKrakendEndpoints(k, list)
	if err != nil {
		return nil, fmt.Errorf("convert ApiEndpoints to Krakend endpoints: %v", err)
	}
	partials, err := json.MarshalIndent(allEndpoints, "", "  ")
	if err != nil {
		return nil, err
	}
	cm.Data[ConfigMapEndpointsKey] = string(partials)
	return cm, nil
}

func resourceName(k *krakendv1.Krakend, suffixes ...string) string {
	name := k.Name + "-" + "gw"
	for _, suffix := range suffixes {
		name += "-" + suffix
	}
	return name
}
