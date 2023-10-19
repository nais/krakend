package webhook

import (
	"github.com/nais/krakend/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"testing"
)

var _ = Describe("ApiEndpoints Validating Webhook", func() {
	var (
		created, fetched, a *v1.ApiEndpoints
		k                   *v1.Krakend
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
				AuthProviders: []v1.AuthProvider{
					{
						Name: "maskinporten",
					},
				},
			},
			Status: v1.KrakendStatus{},
		}
		a = &v1.ApiEndpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing",
				Namespace: "default",
			},
			Spec: newApiEndpointSpec(paths("/before_each_unique1", "/before_each_unique2")),
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

		It("should create an object with unique paths within all apiendpoints objects successfully", func() {

			Expect(k8sClient.Create(ctx, a)).Should(Succeed())

			validMinSpec := newApiEndpointSpec(paths("/unique1", "/unique2"))
			created = apiEndpoints(name, ns, validMinSpec)

			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			fetched = &v1.ApiEndpoints{}
			Eventually(func() error {
				return k8sClient.Get(ctx, nname(created), fetched)
			}).Should(Succeed())

			Expect(k8sClient.Delete(ctx, a)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, created)).Should(Succeed())
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
			existingEndpoints := newApiEndpointSpec(paths("/duplicate", "/unique2"))
			existing := apiEndpoints("existingapp-endpoints", ns, existingEndpoints)
			Expect(k8sClient.Create(ctx, existing)).Should(Succeed())

			validMinSpec := newApiEndpointSpec(paths("/unique1", "/duplicate"))
			created = apiEndpoints(name, ns, validMinSpec)

			Expect(k8sClient.Create(ctx, created)).Should(MatchError(ContainSubstring(MsgPathDuplicate)))
			Expect(k8sClient.Delete(ctx, existing)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, created)).Should(Not(Succeed()))
		})

		It("should fail to create object if auth provider does not exist", func() {
			spec := newApiEndpointSpec(auth("doesnotexist"))
			created = apiEndpoints(name, ns, spec)

			By("creating a valid apiendpoints resource where auth provider does not exist")
			Expect(k8sClient.Create(ctx, created)).Should(Not(Succeed()))
		})

	})
})

func TestUniquePaths(t *testing.T) {
	endpointsList := &v1.ApiEndpointsList{}
	err := parseYaml("testdata/apiendpoints.yaml", endpointsList)
	assert.NoError(t, err)

	up := uniquePaths(endpointsList)
	assert.NoError(t, up)

	err = parseYaml("testdata/apiendpoints_dpaths_diff_app.yaml", endpointsList)
	up = uniquePaths(endpointsList)
	assert.NoError(t, err)
	assert.Error(t, up)

	err = parseYaml("testdata/apiendpoints_dpaths_same_app.yaml", endpointsList)
	up = uniquePaths(endpointsList)
	assert.NoError(t, err)
	assert.Error(t, up)
}

func TestValidateEndpointsList(t *testing.T) {
	apiendpoint := &v1.ApiEndpoints{}
	err := parseYaml("testdata/apiendpoint.yaml", apiendpoint)
	assert.NoError(t, err)

	apiendpointsList := &v1.ApiEndpointsList{}
	err = parseYaml("testdata/apiendpoints.yaml", apiendpointsList)
	assert.NoError(t, err)

	//Validate update of apiendpoint in same apiendpoints resource
	err = validateEndpointsList(apiendpointsList, apiendpoint)
	assert.NoError(t, err)

	//Validate update/create of apiendpoint with duplicate path in a different apiendpoints resource
	err = parseYaml("testdata/apiendpoints_in_other_resource.yaml", apiendpointsList)
	assert.NoError(t, err)
	err = validateEndpointsList(apiendpointsList, apiendpoint)
	assert.Error(t, err)

	//Validate update/create of apiendpoint with unique paths in different apiendpoints resources
	apiendpoint.Spec.Endpoints[0].Path = "/unique"
	err = validateEndpointsList(apiendpointsList, apiendpoint)
	assert.NoError(t, err)

}

func parseYaml(file string, v any) error {
	reader, err := os.Open(file)
	if err != nil {
		return err
	}
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	err = decoder.Decode(v)
	if err != nil {
		return err
	}
	return nil
}

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
		Krakend: "default",
		AppName: "default",
		Auth: v1.Auth{
			Name: "maskinporten",
		},
	}

	if o.Krakend != "" {
		a.Krakend = o.Krakend
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
