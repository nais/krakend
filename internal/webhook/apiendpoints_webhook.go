package webhook

import (
	"context"
	"fmt"
	"github.com/nais/krakend/api/v1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	MsgKrakendDoesNotExist = "the referenced KrakendInstance does not exist"
	MsgPathDuplicate       = "duplicate paths in apiendpoints resource"
)

//+kubebuilder:webhook:path=/validate-apiendpoints,mutating=false,failurePolicy=fail,sideEffects=None,groups=krakend.nais.io,resources=apiendpoints,verbs=create;update,versions=v1,name=apiendpoints.krakend.nais.io,admissionReviewVersions=v1

type ApiEndpointsValidator struct {
	Client  client.Client
	decoder *admission.Decoder
}

// implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (v *ApiEndpointsValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

func (v *ApiEndpointsValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	a := &v1.ApiEndpoints{}
	err := v.decoder.Decode(req, a)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if err := v.validate(ctx, a); err != nil {
		return admission.Denied(err.Error())
	}
	return admission.Allowed("")
}

func (v *ApiEndpointsValidator) validate(ctx context.Context, a *v1.ApiEndpoints) error {
	k := &v1.Krakend{}
	err := v.Client.Get(ctx, types.NamespacedName{
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

	return validateEndpoints(a.Spec.Endpoints)
}

func validateAuth(k *v1.Krakend, auth v1.Auth) error {
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

func validateEndpoints(e []v1.Endpoint) error {
	if len(e) > 0 && e[0].Path == "/duplicate" {
		return fmt.Errorf(MsgPathDuplicate)
	}
	return nil
}
