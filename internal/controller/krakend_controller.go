/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/helm"
	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// KrakendReconciler reconciles a Krakend object
type KrakendReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	SyncInterval time.Duration
	KrakendChart *helm.Chart
}

const DefaultKrakendIngressClass = "nais-ingress-external"

//TODO: add more finegrained permissions

// +kubebuilder:rbac:groups=krakend.nais.io,resources=krakends,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krakend.nais.io,resources=krakends/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=krakend.nais.io,resources=krakends/finalizers,verbs=update
// +kubebuilder:rbac:groups="*",resources=*,verbs=create;update;patch;get;list;watch

func (r *KrakendReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Infof("reconciling krakend %s", req.NamespacedName)
	ns := req.Namespace
	k := &krakendv1.Krakend{}
	err := r.Get(ctx, req.NamespacedName, k)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TODO: add logic for checking if update is neccessary....

	releaseName := k.Spec.Name

	values, err := prepareValues(k)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("preparing values: %w", err)
	}

	resources, err := r.KrakendChart.ToUnstructured(releaseName, chartutil.Values{
		"krakend": values,
	})

	if err != nil {
		return ctrl.Result{}, fmt.Errorf("rendering helm chart: %w", err)
	}

	ownerRef := []metav1.OwnerReference{
		{
			APIVersion: k.APIVersion,
			Kind:       k.Kind,
			Name:       k.Name,
			UID:        k.UID,
		},
	}

	// TODO: IMPORTANT - current logic will overwrite configmap partials containing added api's with default from chart
	for _, resource := range resources {
		log.Debugf("creating resource of kind: %s with name: %s", resource.GetKind(), resource.GetName())

		// TODO: check each resource if any changes are needed, maybe inside createOrUpdate
		// TODO: remove temporary hack for not overwriting configmaps partials
		if resource.GetKind() == "ConfigMap" {
			cmName := fmt.Sprintf("%s-%s-%s", k.Spec.Name, "krakend", "partials")

			cm := &v1.ConfigMap{}
			err := r.Get(ctx, types.NamespacedName{
				Name:      cmName,
				Namespace: ns,
			}, cm)

			if err != nil && !errors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf("get ConfigMap '%s': %v", cmName, err)
			}
			if cm != nil {
				log.Infof("found configmap %s, skipping createOrUpdate", cmName)
				continue
			}
		}

		resource.SetNamespace(ns)
		resource.SetOwnerReferences(ownerRef)
		err := r.createOrUpdate(ctx, resource)
		if err != nil {
			r.Recorder.Eventf(k, "Warning", "CreateResource", "Unable to create resource %v/%v for namespace %q: %v", resource.GetKind(), resource.GetName(), ns, err)
			continue
		}
		log.Debugf("created resource %v/%v for namespace %q", resource.GetKind(), resource.GetName(), ns)
	}

	return ctrl.Result{}, nil
}

func prepareValues(k *krakendv1.Krakend) (map[string]any, error) {
	values, err := toMap(k.Spec.Deployment)
	if err != nil {
		return nil, fmt.Errorf("marshalling krakend deployment: %w", err)
	}

	ingress := k.Spec.Ingress
	ingressHost := k.Spec.IngressHost
	if len(ingress.Hosts) == 0 && ingressHost == "" {
		return nil, fmt.Errorf("either ingressHost or ingress.hosts must be specified")
	}

	if len(ingress.Hosts) == 0 && ingressHost != "" {
		ingress.Hosts = []krakendv1.Host{
			{
				Host: ingressHost,
				Paths: []krakendv1.Path{
					{
						Path:     "/",
						PathType: "ImplementationSpecific",
					},
				},
			},
		}
	}
	ingressValues, err := toMap(ingress)
	if err != nil {
		return nil, fmt.Errorf("preparing ingress values: %w", err)
	}

	values["ingress"] = ingressValues

	return values, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KrakendReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krakendv1.Krakend{}).
		Complete(r)
}

func toMap(v any) (map[string]any, error) {
	j, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	m := make(map[string]any)
	err = json.Unmarshal(j, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *KrakendReconciler) createOrUpdate(
	ctx context.Context,
	resource *unstructured.Unstructured,
) error {
	err := r.Create(ctx, resource)
	if client.IgnoreAlreadyExists(err) != nil {
		return fmt.Errorf("creating resource: %w", err)
	}
	if errors.IsAlreadyExists(err) {
		existing := &unstructured.Unstructured{}
		existing.SetGroupVersionKind(resource.GroupVersionKind())
		err := r.Get(ctx, client.ObjectKeyFromObject(resource), existing)
		if err != nil {
			return fmt.Errorf("getting existing resource: %w", err)
		}
		resource.SetResourceVersion(existing.GetResourceVersion())

		err = r.Update(ctx, resource)
		if err != nil {
			return fmt.Errorf("updating resource: %w", err)
		}
	}
	return nil
}
