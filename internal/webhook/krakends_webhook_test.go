package webhook

import (
	"encoding/json"
	"github.com/nais/krakend/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Krakends Validating Webhook", func() {
	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	// Add Tests for OpenAPI validation (or additonal CRD features) specified in
	// your API definition.
	// Avoid adding tests for vanilla CRUD operations because they would
	// test Kubernetes API server, which isn't the goal here.
	Context("Create Krakends ", func() {
		It("should create an object successfully", func() {
			serviceExtraConfig, _ := json.Marshal(map[string]interface{}{
				"telemetry/opentelemetry": map[string]interface{}{
					"service_name":            "krakend_prometheus_service",
					"metric_reporting_period": 1,
					"exporters": map[string]interface{}{
						"prometheus": []interface{}{
							map[string]interface{}{
								"name":            "local_prometheus",
								"port":            9090,
								"process_metrics": true,
								"go_metrics":      true,
							},
						},
					},
				},
			})

			k := &v1.Krakend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "default",
				},
				Spec: v1.KrakendSpec{
					AuthProviders: []v1.AuthProvider{
						{
							Name: "maskinporten",
						},
					},
					Deployment: v1.KrakendDeployment{
						ExtraConfig: &apiextensionsv1.JSON{
							Raw: serviceExtraConfig,
						},
					},
				},
				Status: v1.KrakendStatus{},
			}
			Expect(k8sClient.Create(ctx, k)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, k)).Should(Succeed())
		})
	})
})
