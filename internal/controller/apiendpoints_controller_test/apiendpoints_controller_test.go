package apiendpoints_test

import (
	"context"
	"fmt"
	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var apiEndpointsDeps *apiEndpointsControllerDependencies

var _ = Describe("ApiEndpoints Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	ctx := context.Background()
	var (
		created *krakendv1.ApiEndpoints
	)

	apiEndpointsDeps = prepareApiEndpointsController()

	BeforeEach(func() {
		Expect(k8sClient.Create(ctx, apiEndpointsDeps.krakend)).Should(Succeed())
		Expect(k8sClient.Create(ctx, apiEndpointsDeps.cm)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, apiEndpointsDeps.krakend)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, apiEndpointsDeps.cm)).Should(Succeed())
	})

	// TODO: add more tests

	Context("Create ApiEndpoints", func() {
		It("should create netpols for endpoints containing apps in same namespace", func() {

			ctx := context.Background()
			created = apiEndpoints("netpoltest", endpoints(
				"http://app1.ns1",
				"http://app2",
				"http://app3.ns1.svc.cluster.local",
				"http://app4.ns2.svc.cluster.local",
				"http://app5.ns2",
				"https://app6.nais.io",
				"invalid",
			))
			expectedNetpols := 3

			actual := &krakendv1.ApiEndpoints{ObjectMeta: created.ObjectMeta}
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())
			Eventually(getApiEndpoints, timeout, interval).WithArguments(k8sClient, ctx, actual).Should(HaveExistingField("SynchronizationHash"))

			np := &networkingv1.NetworkPolicyList{}
			Eventually(func() bool {
				err := k8sClient.List(ctx, np)
				return err == nil && len(np.Items) == expectedNetpols
			}, timeout, interval).Should(BeTrue())
			Expect(np.Items).To(HaveLen(expectedNetpols))

			for _, app := range []string{"app1", "app2", "app3"} {
				n := &networkingv1.NetworkPolicy{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Namespace: actual.Namespace,
					Name:      fmt.Sprintf("%s-%s-%s", "allow", apiEndpointsDeps.krakend.Name, app),
				}, n)).Should(Succeed())
			}

		})
	})
})

type apiEndpointsControllerDependencies struct {
	krakend *krakendv1.Krakend
	cm      *v1.ConfigMap
}

func prepareApiEndpointsController() *apiEndpointsControllerDependencies {
	krakend := &krakendv1.Krakend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ns1",
			Namespace: "ns1",
		},
		Spec: krakendv1.KrakendSpec{
			IngressHost: "ns1.nais.io",
			AuthProviders: []krakendv1.AuthProvider{
				{
					Name:   "authprovider1",
					Alg:    "RS256",
					JwkUrl: "https://jwks",
					Issuer: "https://issuer",
				},
			},
		},
	}
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ns1-krakend-partials",
			Namespace: "ns1",
		},
		Data: map[string]string{
			controller.KrakendConfigMapKey: "[]",
		},
	}

	return &apiEndpointsControllerDependencies{
		krakend: krakend,
		cm:      cm,
	}
}

func getApiEndpoints(c client.Client, ctx context.Context, a *krakendv1.ApiEndpoints) (krakendv1.ApiEndpointsStatus, error) {
	if err := c.Get(ctx, types.NamespacedName{
		Namespace: a.Namespace,
		Name:      a.Name,
	}, a); err != nil {
		return a.Status, err
	}
	return a.Status, nil
}

func apiEndpoints(name string, e []krakendv1.Endpoint) *krakendv1.ApiEndpoints {
	return &krakendv1.ApiEndpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: apiEndpointsDeps.krakend.Namespace,
		},
		Spec: krakendv1.ApiEndpointsSpec{
			Auth: krakendv1.Auth{
				Name: apiEndpointsDeps.krakend.Spec.AuthProviders[0].Name,
			},
			Endpoints: e,
		},
	}
}

func endpoints(backends ...string) []krakendv1.Endpoint {
	e := make([]krakendv1.Endpoint, 0)
	for i, b := range backends {
		e = append(e, krakendv1.Endpoint{
			Path:        fmt.Sprintf("/path%d", i),
			BackendHost: b,
		})
	}
	return e
}
