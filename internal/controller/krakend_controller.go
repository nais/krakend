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
	"fmt"

	hashstructure "github.com/mitchellh/hashstructure/v2"
	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/krakend"
	"github.com/nais/krakend/internal/netpol"
	log "github.com/sirupsen/logrus"
	networkingv1 "k8s.io/api/networking/v1"

	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KrakendReconciler reconciles a Krakend object
type KrakendReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	SyncInterval  time.Duration
	NetpolEnabled bool
}

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

	// write config before the deployment, so new pods have a config to mount
	if err := ensureKrakendConfig(ctx, r.Client, r.Scheme, k); err != nil {
		r.Recorder.Eventf(k, "Warning", "KrakendConfig", "Unable to create krakend config: %v", err)
		return ctrl.Result{}, fmt.Errorf("ensuring krakend config: %w", err)
	}

	objects, warnings, err := krakend.Resources(k)
	if err != nil {
		r.Recorder.Eventf(k, "Warning", "InvalidSpec", "%v", err)
		return ctrl.Result{}, fmt.Errorf("building resources: %w", err)
	}
	for _, w := range warnings {
		r.Recorder.Event(k, "Warning", "InvalidSpec", w)
	}

	ownerRef := []metav1.OwnerReference{
		{
			APIVersion: k.APIVersion,
			Kind:       k.Kind,
			Name:       k.Name,
			UID:        k.UID,
		},
	}

	for _, obj := range objects {
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("converting %T to unstructured: %w", obj, err)
		}
		resource := &unstructured.Unstructured{Object: m}
		resource.SetNamespace(ns)
		resource.SetOwnerReferences(ownerRef)
		if err := r.createOrUpdate(ctx, resource); err != nil {
			r.Recorder.Eventf(k, "Warning", "CreateResource", "Unable to create resource %v/%v for namespace %q: %v", resource.GetKind(), resource.GetName(), ns, err)
			continue
		}
		log.Debugf("created resource %v/%v for namespace %q", resource.GetKind(), resource.GetName(), ns)
	}

	if r.NetpolEnabled {
		if err := r.ensureKrakendNetpol(ctx, k, k.Name); err != nil {
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
