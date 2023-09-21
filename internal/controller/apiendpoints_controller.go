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
	"github.com/nais/krakend/internal/krakend"
	"k8s.io/apimachinery/pkg/types"

	krakendv1 "github.com/nais/krakend/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApiEndpointsReconciler reconciles a ApiEndpoints object
type ApiEndpointsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=krakend.nais.io,resources=apiendpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=krakend.nais.io,resources=apiendpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=krakend.nais.io,resources=apiendpoints/finalizers,verbs=update

func (r *ApiEndpointsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApiEndpointsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krakendv1.ApiEndpoints{}).
		Complete(r)
}

func (r *ApiEndpointsReconciler) updateKrakend(ctx context.Context, endpoints *krakendv1.ApiEndpoints) error {
	name := types.NamespacedName{
		Name:      "cm-partials",
		Namespace: endpoints.Namespace,
	}

	cm := &corev1.ConfigMap{}

	err := r.Get(ctx, name, cm)
	if err != nil {
		return err
	}

	ep := cm.Data["endpoints.tmpl"]
	if ep == "" {
		return fmt.Errorf("endpoints.tmpl not found in ConfigMap")
	}

	existing := &krakend.Partials{}
	err = json.Unmarshal([]byte(ep), &existing.Endpoints)
	if err != nil {
		return err
	}

	n := krakend.ParseKrakendEndpointsSpec(endpoints.Spec)
	merged, err := krakend.MergePartials(existing, n)
	partials, err := json.Marshal(merged.Endpoints)
	if err != nil {
		return err
	}

	//TODO handle race conditions when updating configmap
	cm.Data["endpoints.tmpl"] = string(partials)
	err = r.Update(ctx, cm)
	if err != nil {
		return err
	}

	return nil
}
