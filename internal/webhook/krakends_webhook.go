package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/santhosh-tekuri/jsonschema/v5"
	_ "github.com/santhosh-tekuri/jsonschema/v5/httploader"
	log "github.com/sirupsen/logrus"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-krakends,mutating=false,failurePolicy=fail,sideEffects=None,groups=krakend.nais.io,resources=krakends,verbs=create;update,versions=v1,name=krakends.krakend.nais.io,admissionReviewVersions=v1

type KrakendsValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

func (v *KrakendsValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	log.Infof("registering webhook server at /validate-apiendpoints")
	v.decoder = admission.NewDecoder(mgr.GetScheme())
	v.client = mgr.GetClient()
	mgr.GetWebhookServer().Register("/validate-krakends", &webhook.Admission{Handler: v})
	return nil
}

func (v *KrakendsValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	k := &krakendv1.Krakend{}
	err := v.decoder.Decode(req, k)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if err := v.validate(ctx, k); err != nil {
		return admission.Denied(err.Error())
	}
	return admission.Allowed("")
}

func (v *KrakendsValidator) validate(ctx context.Context, k *krakendv1.Krakend) error {
	if k.Spec.Deployment.ExtraConfig != nil {
		var serviceExtraConfig interface{}
		if err := json.Unmarshal(k.Spec.Deployment.ExtraConfig.Raw, &serviceExtraConfig); err != nil {
			return fmt.Errorf("unmarshaling serviceExtraConfig: %w", err)
		}

		sch, err := jsonschema.Compile("https://www.krakend.io/schema/v2.7/service_extra_config.json")
		if err != nil {
			return fmt.Errorf("compling serviceExtraConfig json schema: %w", err)
		}

		if err = sch.Validate(serviceExtraConfig); err != nil {
			return fmt.Errorf("linting the serviceExtraConfig: %w", err)
		}
	}

	return nil
}
