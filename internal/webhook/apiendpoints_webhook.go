package webhook

import (
	"context"
	"fmt"
	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/utils"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	MsgKrakendDoesNotExist = "the referenced KrakendInstance does not exist"
)

//+kubebuilder:webhook:path=/validate-apiendpoints,mutating=false,failurePolicy=fail,sideEffects=None,groups=krakend.nais.io,resources=apiendpoints,verbs=create;update,versions=v1,name=apiendpoints.krakend.nais.io,admissionReviewVersions=v1

type ApiEndpointsValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

func (v *ApiEndpointsValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	log.Infof("registering webhook server at /validate-apiendpoints")
	v.decoder = admission.NewDecoder(mgr.GetScheme())
	v.client = mgr.GetClient()
	mgr.GetWebhookServer().Register("/validate-apiendpoints", &webhook.Admission{Handler: v})
	return nil
}

func (v *ApiEndpointsValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	a := &krakendv1.ApiEndpoints{}
	err := v.decoder.Decode(req, a)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if err := v.validate(ctx, a); err != nil {
		return admission.Denied(err.Error())
	}
	return admission.Allowed("")
}

func (v *ApiEndpointsValidator) validate(ctx context.Context, a *krakendv1.ApiEndpoints) error {
	k := &krakendv1.Krakend{}
	err := v.client.Get(ctx, types.NamespacedName{
		Name:      a.Spec.KrakendInstance,
		Namespace: a.Namespace,
	}, k)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("getting krakendinstance: %w", err)
	}
	if errors.IsNotFound(err) {
		return fmt.Errorf(MsgKrakendDoesNotExist)
	}
	log.Infof("found krakendinstance %s", k.Name)

	err = validateAuth(k, a.Spec.Auth)
	if err != nil {
		return err
	}

	el := &krakendv1.ApiEndpointsList{}
	err = v.client.List(ctx, el, client.InNamespace(k.Namespace))
	if err != nil {
		return fmt.Errorf("getting list of apiendpoints: %w", err)
	}
	return utils.ValidateEndpointsList(el, a)
}

func validateAuth(k *krakendv1.Krakend, auth krakendv1.Auth) error {
	found := false
	for _, p := range k.Spec.AuthProviders {
		if p.Name == auth.Name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("auth provider %s not found in krakendinstance %s", auth.Name, k.Name)
	}
	return nil
}
