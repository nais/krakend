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
)

// ApiEndpointsReconciler reconciles a ApiEndpoints object
type ApiEndpointsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	AppLabelName = "app"
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

	k := &krakendv1.Krakend{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      endpoints.Spec.KrakendInstance,
		Namespace: endpoints.Namespace,
	}, k)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("get Krakend instance '%s': %v", endpoints.Spec.KrakendInstance, err)
	}

	ownerRef := []metav1.OwnerReference{
		{
			APIVersion: k.APIVersion,
			Kind:       k.Kind,
			Name:       k.Name,
			UID:        k.UID,
		},
	}
	endpoints.SetOwnerReferences(ownerRef)

	if err := r.updateKrakendConfigMap(ctx, k, endpoints); err != nil {
		log.Errorf("updating Krakend configmap: %v", err)
		return ctrl.Result{}, nil
	}

	//TODO: check sync hash and skip if unchanged
	if err := r.ensureAppIngressNetpol(ctx, endpoints); err != nil {
		log.Errorf("creating/updating netpol: %v", err)
		return ctrl.Result{}, nil
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
	// TODO: only create if app (in ApiEndpoints) is in same namespace, e.g. skip if host is outside cluster
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
	err := r.Get(ctx, types.NamespacedName{
		Name:      npName,
		Namespace: endpoints.Namespace,
	}, np)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	if errors.IsNotFound(err) {
		np = netpol.AllowKrakendIngressNetpol(npName, endpoints.Namespace, map[string]string{
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
func (r *ApiEndpointsReconciler) updateKrakendConfigMap(ctx context.Context, k *krakendv1.Krakend, endpoints *krakendv1.ApiEndpoints) error {
	cm := &corev1.ConfigMap{}
	cmName := fmt.Sprintf("%s-%s-%s", k.Spec.Name, "krakend", "partials")
	err := r.Get(ctx, types.NamespacedName{
		Name:      cmName,
		Namespace: endpoints.Namespace,
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
	if err = r.List(ctx, list, client.InNamespace(endpoints.Namespace)); err != nil {
		return fmt.Errorf("list all ApiEndpoints: %v", err)
	}

	allEndpoints, err := krakend.ToKrakendEndpoints(k, list)
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
