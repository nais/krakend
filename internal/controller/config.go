package controller

import (
	"context"
	"fmt"

	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/krakend"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureKrakendConfig renders krakend.json from the Krakend resource and all ApiEndpoints in its
// namespace, and creates or updates the ConfigMap if the content differs.
func ensureKrakendConfig(ctx context.Context, c client.Client, scheme *runtime.Scheme, k *krakendv1.Krakend) error {
	list := &krakendv1.ApiEndpointsList{}
	if err := c.List(ctx, list, client.InNamespace(k.Namespace)); err != nil {
		return fmt.Errorf("list ApiEndpoints: %w", err)
	}

	// all ApiEndpoints in the namespace are included, regardless of spec.krakend (same as before v3)
	items := make([]krakendv1.ApiEndpoints, 0, len(list.Items))
	for _, e := range list.Items {
		if e.GetDeletionTimestamp() == nil {
			items = append(items, e)
		}
	}

	cfg, err := krakend.BuildConfig(k, items)
	if err != nil {
		return fmt.Errorf("build krakend config: %w", err)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      krakend.ConfigMapName(k),
			Namespace: k.Namespace,
		},
	}
	err = c.Get(ctx, client.ObjectKeyFromObject(cm), cm)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("get ConfigMap %q: %w", cm.Name, err)
	}
	exists := err == nil

	if exists && cm.Data[krakend.ConfigFileKey] == string(cfg) {
		return nil
	}

	cm.Data = map[string]string{krakend.ConfigFileKey: string(cfg)}
	if cm.Labels == nil {
		cm.Labels = map[string]string{}
	}
	cm.Labels["app.kubernetes.io/name"] = "krakend"
	cm.Labels["app.kubernetes.io/instance"] = k.Name
	cm.Labels["app.kubernetes.io/managed-by"] = "krakend-operator"
	if cm.Annotations == nil {
		cm.Annotations = map[string]string{}
	}
	cm.Annotations["reloader.stakater.com/match"] = "true"
	if err := controllerutil.SetOwnerReference(k, cm, scheme); err != nil {
		return fmt.Errorf("set owner reference: %w", err)
	}

	if exists {
		if err := c.Update(ctx, cm); err != nil {
			return fmt.Errorf("update ConfigMap %q: %w", cm.Name, err)
		}
		log.Infof("updated ConfigMap %s/%s", cm.Namespace, cm.Name)
		return nil
	}
	if err := c.Create(ctx, cm); err != nil {
		return fmt.Errorf("create ConfigMap %q: %w", cm.Name, err)
	}
	log.Infof("created ConfigMap %s/%s", cm.Namespace, cm.Name)
	return nil
}
