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
	"github.com/mitchellh/hashstructure/v2"
	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/krakend"
	"github.com/nais/krakend/internal/netpol"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

// ApiEndpointsReconciler reconciles a ApiEndpoints object
type ApiEndpointsReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	SyncInterval  time.Duration
	NetpolEnabled bool
}

const (
	AppLabelName     = "app"
	KrakendFinalizer = "finalizer.krakend.nais.io"
)

//+kubebuilder:rbac:groups=krakend.nais.io,resources=apiendpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=krakend.nais.io,resources=apiendpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=krakend.nais.io,resources=apiendpoints/finalizers,verbs=update

func (r *ApiEndpointsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.WithFields(log.Fields{
		"apiendpoints_name":      req.Name,
		"apiendpoints_namespace": req.Namespace,
	}).Infof("Reconciling ApiEndpoints")

	endpoints := &krakendv1.ApiEndpoints{}
	if err := r.Get(ctx, req.NamespacedName, endpoints); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if endpoints.GetDeletionTimestamp() != nil {
		log.Debugf("Resource %s is marked for deletion", endpoints.Name)

		k := &krakendv1.Krakend{}
		err := r.Get(ctx, types.NamespacedName{
			Name:      endpoints.Spec.KrakendInstance,
			Namespace: endpoints.Namespace,
		}, k)
		if err != nil {
			if !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			log.Debugf("krakend '%s' not found, nothing to do but remove finalizers", endpoints.Spec.KrakendInstance)
		} else {
			err = r.updateKrakendConfigMap(ctx, k)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		if controllerutil.RemoveFinalizer(endpoints, KrakendFinalizer) {
			err := r.Update(ctx, endpoints)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	hash, err := hashEndpoints(endpoints.Spec)
	if err != nil {
		return ctrl.Result{}, err
	}

	// skip reconciliation if hash is unchanged and timestamp is within sync interval
	// reconciliation is triggered when status subresource is updated, so we need this check to avoid infinite loop
	if endpoints.Status.SynchronizationHash == hash && !r.needsSync(endpoints.Status.SynchronizationTimestamp.Time) {
		log.Debugf("skipping reconciliation of %q, hash %q is unchanged and changed within syncInterval window", endpoints.Name, hash)
		return ctrl.Result{}, nil
	} else {
		log.Debugf("reconciling: hash changed: %v, outside syncInterval window: %v", endpoints.Status.SynchronizationHash != hash, r.needsSync(endpoints.Status.SynchronizationTimestamp.Time))
	}

	k := &krakendv1.Krakend{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      endpoints.Spec.KrakendInstance,
		Namespace: endpoints.Namespace,
	}, k)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("get Krakend instance '%s': %v", endpoints.Spec.KrakendInstance, err)
	}
	err = r.updateKrakendConfigMap(ctx, k)
	if err != nil {
		log.Errorf("updating Krakend configmap: %v", err)
		return ctrl.Result{}, err
	}

	if r.NetpolEnabled {
		if err := r.ensureAppIngressNetpol(ctx, endpoints); err != nil {
			log.Errorf("creating/updating netpol: %v", err)
			return ctrl.Result{}, nil
		}
	}

	needsUpdate := controllerutil.AddFinalizer(endpoints, KrakendFinalizer)
	if endpoints.GetOwnerReferences() == nil {
		ownerRef := []metav1.OwnerReference{
			{
				APIVersion: k.APIVersion,
				Kind:       k.Kind,
				Name:       k.Name,
				UID:        k.UID,
			},
		}

		endpoints.SetOwnerReferences(ownerRef)
		needsUpdate = true
	}

	if needsUpdate {
		if err := r.Update(ctx, endpoints); err != nil {
			return ctrl.Result{}, err
		}
	}

	endpoints.Status.SynchronizationTimestamp = metav1.Now()
	endpoints.Status.SynchronizationHash = hash
	if err := r.Status().Update(ctx, endpoints); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApiEndpointsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krakendv1.ApiEndpoints{}).
		Complete(r)
}

func (r *ApiEndpointsReconciler) ensureAppIngressNetpol(ctx context.Context, endpoints *krakendv1.ApiEndpoints) error {
	// TODO: use label selector instead of app name
	svc := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      endpoints.Spec.AppName,
		Namespace: endpoints.Namespace,
	}, svc)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	if errors.IsNotFound(err) {
		log.Debugf("service for app %s not found, skipping ingress netpol", endpoints.Spec.AppName)
		return nil
	}

	ownerRef := []metav1.OwnerReference{
		{
			APIVersion: endpoints.APIVersion,
			Kind:       endpoints.Kind,
			Name:       endpoints.Name,
			UID:        endpoints.UID,
		},
	}

	npName := fmt.Sprintf("%s-%s-%s", "allow", endpoints.Spec.KrakendInstance, endpoints.Spec.AppName)

	np := &v1.NetworkPolicy{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      npName,
		Namespace: endpoints.Namespace,
	}, np)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	if errors.IsNotFound(err) {
		np = netpol.AppAllowKrakendIngressNetpol(npName, endpoints.Namespace, map[string]string{
			AppLabelName: endpoints.Spec.AppName,
		})
		np.SetOwnerReferences(ownerRef)

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

// TODO: validate unique paths - maybe webhook?
func (r *ApiEndpointsReconciler) updateKrakendConfigMap(ctx context.Context, k *krakendv1.Krakend) error {

	cm := &corev1.ConfigMap{}
	cmName := fmt.Sprintf("%s-%s-%s", k.Spec.Name, "krakend", "partials")
	err := r.Get(ctx, types.NamespacedName{
		Name:      cmName,
		Namespace: k.Namespace,
	}, cm)
	if err != nil {
		return fmt.Errorf("get ConfigMap '%s': %v", cmName, err)
	}

	key := "endpoints.tmpl"
	ep := cm.Data[key]
	if ep == "" {
		return fmt.Errorf("%s not found in ConfigMap with name %s", key, cmName)
	}

	list := &krakendv1.ApiEndpointsList{}
	if err = r.List(ctx, list, client.InNamespace(k.Namespace)); err != nil {
		return fmt.Errorf("list all ApiEndpoints: %v", err)
	}

	if err := UniquePaths(list); err != nil {
		return fmt.Errorf("validate unique paths: %v", err)
	}

	filtered := make([]krakendv1.ApiEndpoints, 0)
	for _, e := range list.Items {
		if e.GetDeletionTimestamp() == nil {
			filtered = append(filtered, e)
		}
	}

	allEndpoints, err := krakend.ToKrakendEndpoints(k, filtered)
	if err != nil {
		return fmt.Errorf("convert ApiEndpoints to Krakend endpoints: %v", err)
	}
	partials, err := json.Marshal(allEndpoints)
	if err != nil {
		return err
	}

	//TODO handle race conditions when updating configmap
	cm.Data[key] = string(partials)
	err = r.Update(ctx, cm)
	if err != nil {
		return fmt.Errorf("update ConfigMap '%s': %v", cmName, err)
	}

	return nil
}

func UniquePaths(list *krakendv1.ApiEndpointsList) error {

	paths := make(map[string]string)
	for _, e := range list.Items {
		if e.GetDeletionTimestamp() == nil {
			if len(e.Spec.Endpoints) > 0 {
				for _, p := range e.Spec.Endpoints {
					if _, ok := paths[p.Path]; ok {
						log.Warnf("duplicate path %s in endpoints %s and %s", p.Path, e.Name, paths[p.Path])
						return fmt.Errorf("duplicate path %s in endpoints %s and %s", p.Path, e.Name, paths[p.Path])
					} else {
						paths[p.Path] = e.Name
					}
				}
			}
			if len(e.Spec.OpenEndpoints) > 0 {
				for _, p := range e.Spec.OpenEndpoints {
					if _, ok := paths[p.Path]; ok {
						log.Warnf("duplicate path %s in openEndpoints %s and %s", p.Path, e.Name, paths[p.Path])
						return fmt.Errorf("duplicate path %s in endpoints %s and %s", p.Path, e.Name, paths[p.Path])
					} else {
						paths[p.Path] = e.Name
					}
				}
			}
		}
	}
	return nil
}

func hashEndpoints(a krakendv1.ApiEndpointsSpec) (string, error) {
	hash, err := hashstructure.Hash(a, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash), nil
}

func (r *ApiEndpointsReconciler) needsSync(timestamp time.Time) bool {
	window := time.Now().Add(-r.SyncInterval)
	return timestamp.Before(window)
}
