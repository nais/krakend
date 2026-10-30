package parse

import (
	"fmt"

	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/krakend"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ToKrakendConfig returns a single ConfigMap with the complete krakend.json, same format as the operator.
func ToKrakendConfig(k *krakendv1.Krakend, list []krakendv1.ApiEndpoints) (*corev1.ConfigMap, error) {
	cfg, err := krakend.BuildConfig(k, list)
	if err != nil {
		return nil, fmt.Errorf("build krakend config: %w", err)
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName(k, "config"),
			Namespace: k.Namespace,
			Annotations: map[string]string{
				"reloader.stakater.com/match": "true",
			},
		},
		Data: map[string]string{
			krakend.ConfigFileKey: string(cfg),
		},
	}, nil
}

func resourceName(k *krakendv1.Krakend, suffixes ...string) string {
	name := k.Name + "-" + "gw"
	for _, suffix := range suffixes {
		name += "-" + suffix
	}
	return name
}
