package v1

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ApiEndpoints Webhook", func() {

	const timeout = time.Second * 30
	const interval = time.Second * 1

	const name = "apiendpoints-minimal"

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
	Context("ApiEndpoints", func() {
		It("Should handle ApiEndpoints", func() {
			spec := ApiEndpointsSpec{
				KrakendInstance: "",
				AppName:         "",
				Auth:            Auth{},
				Endpoints:       nil,
				OpenEndpoints:   nil,
			}

			key := types.NamespacedName{
				Name:      name,
				Namespace: "default",
			}

			toCreate := &ApiEndpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			err := k8sClient.Create(context.Background(), toCreate)
			if err != nil {
				fmt.Println(err)
			}
		})
	})
})
