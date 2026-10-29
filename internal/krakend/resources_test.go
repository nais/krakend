package krakend

import (
	"os"
	"testing"

	v1 "github.com/nais/krakend/api/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResources(t *testing.T) {
	k := &v1.Krakend{
		ObjectMeta: metav1.ObjectMeta{Name: "team1", Namespace: "team1"},
		Spec: v1.KrakendSpec{
			IngressHost: "team1.nais.io",
			Deployment: v1.KrakendDeployment{
				Image: v1.Image{Repository: "devopsfaith/krakend", Tag: "2.4.3"},
			},
		},
	}

	objs, warnings, err := Resources(k)
	assert.NoError(t, err)
	assert.Len(t, warnings, 1, "image tag 2.4.3 should be ignored")
	assert.Len(t, objs, 5)

	for _, o := range objs {
		assert.Equal(t, "team1-krakend", o.GetName(), "names must stay compatible with existing installations")
		switch r := o.(type) {
		case *appsv1.Deployment:
			assert.Equal(t, int32(2), *r.Spec.Replicas)
			assert.Equal(t, map[string]string{
				"app.kubernetes.io/name":     "krakend",
				"app.kubernetes.io/instance": "team1",
			}, r.Spec.Selector.MatchLabels, "selector is immutable on existing deployments")
			assert.Equal(t, ImageRepository+":"+ImageTag, r.Spec.Template.Spec.Containers[0].Image)
			assert.Equal(t, "team1-krakend-v3", r.Spec.Template.Spec.Volumes[1].ConfigMap.Name)
		case *networkingv1.Ingress:
			assert.Equal(t, "team1.nais.io", r.Spec.Rules[0].Host)
			assert.Equal(t, "external-haproxy", *r.Spec.IngressClassName)
		}
	}
}

func TestImageTagMatchesDockerfile(t *testing.T) {
	b, err := os.ReadFile("../../krakend/Dockerfile")
	assert.NoError(t, err)
	assert.Contains(t, string(b), "\nFROM krakend:"+ImageTag+"@sha256:")
}

func TestResourcesRequiresIngressHost(t *testing.T) {
	_, _, err := Resources(&v1.Krakend{ObjectMeta: metav1.ObjectMeta{Name: "team1"}})
	assert.Error(t, err)
}

func TestSupportedImageTag(t *testing.T) {
	for tag, expected := range map[string]bool{
		"3.0.2": true, "3": true, "v3.1": true, "4.0.0": true,
		"2.13.10": false, "2.4.3": false, "latest": false, "": false,
	} {
		assert.Equalf(t, expected, SupportedImageTag(tag), "tag %q", tag)
	}
}

func TestFullname(t *testing.T) {
	assert.Equal(t, "team1-krakend", Fullname(&v1.Krakend{ObjectMeta: metav1.ObjectMeta{Name: "team1"}}))
	assert.Equal(t, "team1-krakend-gw", Fullname(&v1.Krakend{ObjectMeta: metav1.ObjectMeta{Name: "team1-krakend-gw"}}))
}
