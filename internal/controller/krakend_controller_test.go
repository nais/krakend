package controller

import (
	"context"
	"fmt"
	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/helm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

func getKrakend(c client.Client, ctx context.Context, krakend *krakendv1.Krakend) (krakendv1.KrakendStatus, error) {
	if err := c.Get(ctx, krakend.NamespacedName(), krakend); err != nil {
		return krakend.Status, err
	}
	return krakend.Status, nil
}

func krakendResource(ns, name string, spec krakendv1.KrakendSpec) *krakendv1.Krakend {
	return &krakendv1.Krakend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: spec,
	}
}

func fullKrakendSpec(name string) krakendv1.KrakendSpec {
	return krakendv1.KrakendSpec{
		Name:          name,
		IngressHost:   "krakend.nais.io",
		AuthProviders: []krakendv1.AuthProvider{},
		Deployment: krakendv1.KrakendDeployment{
			DeploymentType: "deployment",
			ReplicaCount:   1,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("200m"),
					"memory": resource.MustParse("128Mi"),
				},
			},
			Image: krakendv1.Image{
				Repository: "nais.io",
				Tag:        "greatest",
				PullPolicy: "Always",
			},
			ExtraEnvVars: []corev1.EnvVar{
				{
					Name:  "MY_ENV_VAR",
					Value: "MY_ENV_VAR_VALUE",
				},
			},
		},
	}
}

var _ = Describe("Krakend Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var (
		created *krakendv1.Krakend
	)

	BeforeEach(func() {

	})

	AfterEach(func() {

	})

	// Add Tests for OpenAPI validation (or additonal CRD features) specified in
	// your API definition.
	// Avoid adding tests for vanilla CRUD operations because they would
	// test Kubernetes API server, which isn't the goal here.
	Context("Create Krakend", func() {
		It("should create an Krakend installation successfully", func() {

			ctx := context.Background()
			created = krakendResource("default", "team1", fullKrakendSpec("team1"))

			actual := &krakendv1.Krakend{ObjectMeta: created.ObjectMeta}
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())
			Eventually(getKrakend, timeout, interval).WithArguments(k8sClient, ctx, actual).Should(HaveExistingField("SynchronizationHash"))

			d := &v1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: "default",
					Name:      "team1-krakend",
				}, d)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})

func TestRenderChart(t *testing.T) {
	k, err := unmarshallKrakend("testdata/krakend_min.yaml")
	assert.NoError(t, err)

	values, err := prepareValues(k)
	assert.NoError(t, err)

	c, err := helm.LoadChart("testdata/krakend")
	assert.NoError(t, err)

	resources, err := c.ToUnstructured(k.Spec.Name, chartutil.Values{
		"krakend": values,
	})
	assert.NoError(t, err)

	for _, r := range resources {
		fmt.Printf("%+v\n", r)
		if r.GetKind() == "Deployment" {
			assert.Equal(t, 2, r.Object["spec"].(map[string]interface{})["replicas"])
			return
		}
	}
}

func unmarshallKrakend(yamlFile string) (*krakendv1.Krakend, error) {
	sch := runtime.NewScheme()
	_ = scheme.AddToScheme(sch)
	err := krakendv1.AddToScheme(sch)
	if err != nil {
		return nil, err
	}
	_ = apiextv1beta1.AddToScheme(sch)
	decode := serializer.NewCodecFactory(sch).UniversalDeserializer().Decode
	stream, _ := os.ReadFile(yamlFile)
	obj, gvk, err := decode(stream, nil, nil)
	if err != nil {
		return nil, err
	}

	if gvk.Kind == "Krakend" {
		return obj.(*krakendv1.Krakend), nil
	}
	return nil, fmt.Errorf("kind is not krakend")
}
