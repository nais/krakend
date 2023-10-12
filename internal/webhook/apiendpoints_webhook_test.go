package webhook

import (
	"github.com/nais/krakend/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ApiEndpoints Validating Webhook", func() {
	var (
		created, fetched *v1.ApiEndpoints
		k                *v1.Krakend
	)

	name := "valid"
	ns := "default"

	BeforeEach(func() {
		k = &v1.Krakend{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: "default",
			},
			Spec: v1.KrakendSpec{
				Name: "default",
				AuthProviders: []v1.AuthProvider{
					{
						Name: "maskinporten",
					},
				},
			},
			Status: v1.KrakendStatus{},
		}
		// Add any setup steps that needs to be executed before each test
		Expect(k8sClient.Create(ctx, k)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, k)).Should(Succeed())
		// Add any teardown steps that needs to be executed after each test
	})

	// Add Tests for OpenAPI validation (or additonal CRD features) specified in
	// your API definition.
	// Avoid adding tests for vanilla CRUD operations because they would
	// test Kubernetes API server, which isn't the goal here.
	Context("Create ApiEndpoints ", func() {

		It("should create an object successfully", func() {
			validMinSpec := newApiEndpointSpec(paths("/unique1", "/unique2"))
			created = apiEndpoints(name, ns, validMinSpec)

			By("creating a valid apiendpoints resource with unique paths")
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			fetched = &v1.ApiEndpoints{}
			Eventually(func() error {
				return k8sClient.Get(ctx, nname(created), fetched)
			}).Should(Succeed())

			err := k8sClient.Delete(ctx, fetched)
			Expect(err).Should(Succeed())

		})

		It("should fail to create an object with duplicate paths in same object", func() {
			duplicatePaths := newApiEndpointSpec(paths("/duplicate", "/duplicate"))
			created = apiEndpoints(name, ns, duplicatePaths)

			By("creating an valid apiendpoints resource with duplicate paths")
			Expect(k8sClient.Create(ctx, created)).Should(MatchError(ContainSubstring(MsgPathDuplicate)))

		})

		It("should fail to create if krakendinstance does not exist", func() {
			spec := newApiEndpointSpec(krakend("doesnotexist"))
			created = apiEndpoints(name, ns, spec)

			By("creating a valid apiendpoints resource where krakendinstance does not exist")
			Expect(k8sClient.Create(ctx, created)).Should(MatchError(ContainSubstring(MsgKrakendDoesNotExist)))
		})

		It("should fail to create an object with duplicate paths within all apiendpoints objects", func() {
			// TODO
		})

		It("should fail to create object if auth provider does not exist", func() {
			spec := newApiEndpointSpec(auth("doesnotexist"))
			created = apiEndpoints(name, ns, spec)

			By("creating a valid apiendpoints resource where auth provider does not exist")
			Expect(k8sClient.Create(ctx, created)).Should(Not(Succeed()))
		})

	})
})

func apiEndpoints(name, namespace string, spec v1.ApiEndpointsSpec) *v1.ApiEndpoints {
	return &v1.ApiEndpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
	}
}

type options struct {
	Paths   []string
	Krakend string
	App     string
	Auth    string
}

type option func(*options)

func paths(paths ...string) option {
	return func(o *options) {
		o.Paths = paths
	}
}

func krakend(krakend string) option {
	return func(o *options) {
		o.Krakend = krakend
	}
}

func auth(auth string) option {
	return func(o *options) {
		o.Auth = auth
	}
}

func newApiEndpointSpec(opts ...option) v1.ApiEndpointsSpec {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	a := v1.ApiEndpointsSpec{
		KrakendInstance: "default",
		AppName:         "default",
		Auth: v1.Auth{
			Name: "maskinporten",
		},
	}

	if o.Krakend != "" {
		a.KrakendInstance = o.Krakend
	}
	if o.App != "" {
		a.AppName = o.App
	}
	if o.Auth != "" {
		a.Auth = v1.Auth{
			Name: o.Auth,
		}
	}

	if len(o.Paths) > 0 {
		for _, path := range o.Paths {
			a.Endpoints = append(a.Endpoints, v1.Endpoint{
				Path:        path,
				Method:      "GET",
				BackendHost: "http://host1",
				BackendPath: "/path",
			})
		}
	}
	return a
}

func nname(a *v1.ApiEndpoints) types.NamespacedName {
	return types.NamespacedName{
		Name:      a.Name,
		Namespace: a.Namespace,
	}
}
