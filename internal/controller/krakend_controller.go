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
	hashstructure "github.com/mitchellh/hashstructure/v2"
	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/helm"
	"github.com/nais/krakend/internal/netpol"
	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"strings"

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
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	SyncInterval  time.Duration
	KrakendChart  *helm.Chart
	NetpolEnabled bool
}

const DefaultKrakendIngressClass = "nais-ingress-external"

//TODO: add more finegrained permissions

// +kubebuilder:rbac:groups=krakend.nais.io,resources=krakends,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krakend.nais.io,resources=krakends/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=krakend.nais.io,resources=krakends/finalizers,verbs=update
// +kubebuilder:rbac:groups="*",resources=*,verbs=create;update;patch;get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=create;update;patch;get;list;watch;delete

func (r *KrakendReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Infof("reconciling krakend %s", req.NamespacedName)
	ns := req.Namespace
	k := &krakendv1.Krakend{}
	err := r.Get(ctx, req.NamespacedName, k)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	hash, err := hash(k.Spec)
	if err != nil {
		return ctrl.Result{}, err
	}

	// skip reconciliation if hash is unchanged and timestamp is within sync interval
	// reconciliation is triggered when status subresource is updated, so we need this check to avoid infinite loop
	if k.Status.SynchronizationHash == hash && !r.needsSync(k.Status.SynchronizationTimestamp.Time) {
		log.Debugf("skipping reconciliation of %q, hash %q is unchanged and changed within syncInterval window", k.Name, hash)
		return ctrl.Result{}, nil
	} else {
		log.Debugf("reconciling: hash changed: %v, outside syncInterval window: %v", k.Status.SynchronizationHash != hash, r.needsSync(k.Status.SynchronizationTimestamp.Time))
	}

	releaseName := k.Name
	releaseNamespace := k.Namespace

	values, err := prepareValues(k)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("preparing values: %w", err)
	}

	resources, err := r.KrakendChart.ToUnstructured(releaseName, releaseNamespace, chartutil.Values{
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

	for _, resource := range resources {
		log.Debugf("creating resource of kind: %s with name: %s", resource.GetKind(), resource.GetName())

		if resource.GetKind() == "Deployment" {
			addAnnotations(resource, map[string]string{"reloader.stakater.com/search": "true"})
			d := &v1.Deployment{}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, d)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("converting unstructured to deployment: %w", err)
			}

			d.Spec.Template.Name = d.Name
			if len(d.Spec.Template.Spec.Containers) == 1 {
				d.Spec.Template.Spec.Containers[0].Name = d.Name
				existing := d.Spec.Template.Spec.Containers[0].Env
				existing = append(existing, k.Spec.Deployment.ExtraEnvVars...)
				d.Spec.Template.Spec.Containers[0].Env = existing
			}
			d.Spec.Template.Annotations["kubectl.kubernetes.io/default-container"] = d.Name

			m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(d)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("converting deployment to unstructured: %w", err)
			}
			resource.Object = m
		}

		if resource.GetKind() == "ConfigMap" {
			addAnnotations(resource, map[string]string{"reloader.stakater.com/match": "true"})

			var cm corev1.ConfigMap
			err := r.Get(ctx, types.NamespacedName{
				Name:      resource.GetName(),
				Namespace: ns,
			}, &cm)

			if err != nil && !errors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf("get ConfigMap '%s': %v", resource.GetName(), err)
			}
			// skip updating partials configmap if it already exists
			// to avoid deleting existing apiendpoints
			if err == nil && strings.HasSuffix(resource.GetName(), "-partials") {
				log.Infof("found configmap %s, skipping createOrUpdate", resource.GetName())
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

	if r.NetpolEnabled {
		if err := r.ensureKrakendNetpol(ctx, k, releaseName); err != nil {
			return ctrl.Result{}, fmt.Errorf("ensuring krakend egress netpol: %w", err)
		}
	}

	k.Status.SynchronizationTimestamp = metav1.Now()
	k.Status.SynchronizationHash = hash
	if err := r.Status().Update(ctx, k); err != nil {
		r.Recorder.Eventf(k, "Warning", "UpdateStatus", "Unable to update status for %q: %v", k.Name, err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func addAnnotations(resource *unstructured.Unstructured, annotations map[string]string) {
	existing := resource.GetAnnotations()
	if existing == nil {
		existing = make(map[string]string)
	}
	for k, v := range annotations {
		existing[k] = v
	}
	resource.SetAnnotations(existing)
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
	values["krakend"] = map[string]interface{}{
		"extraConfig": values["extraConfig"],
	}

	return values, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KrakendReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krakendv1.Krakend{}).
		Complete(r)
}

// TODO: this is temporary set to allow egress to all IPs and not per endpoint, consider creating fqdn policy for each endpoint. If we choose to do this, move this function to apiendpoints controller instead.
func (r *KrakendReconciler) ensureKrakendNetpol(ctx context.Context, k *krakendv1.Krakend, releaseName string) error {
	ownerRef := []metav1.OwnerReference{
		{
			APIVersion: k.APIVersion,
			Kind:       k.Kind,
			Name:       k.Name,
			UID:        k.UID,
		},
	}

	npName := fmt.Sprintf("%s-%s", releaseName, "krakend")

	existing := &networkingv1.NetworkPolicy{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      npName,
		Namespace: k.Namespace,
	}, existing)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	np := netpol.KrakendNetpol(npName, k.Namespace, map[string]string{
		// TODO: some logic to get the correct label?
		"app.kubernetes.io/name": "krakend",
	})
	np.SetOwnerReferences(ownerRef)

	if errors.IsNotFound(err) {

		err := r.Create(ctx, np)
		if err != nil {
			return fmt.Errorf("create netpol: %v", err)
		}
		return nil
	}

	//TODO: diff and update if needed
	err = r.Update(ctx, np)
	if err != nil {
		return fmt.Errorf("update netpol: %v", err)
	}
	return nil
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

// TODO: diff and update - see nais/replicator for inspiration
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

func hash(k krakendv1.KrakendSpec) (string, error) {
	hash, err := hashstructure.Hash(k, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash), nil
}

func (r *KrakendReconciler) needsSync(timestamp time.Time) bool {
	window := time.Now().Add(-r.SyncInterval)
	return timestamp.Before(window)
}
