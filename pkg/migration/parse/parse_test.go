package parse

import (
	"strings"
	"testing"

	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/nais/krakend/internal/krakend"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvert(t *testing.T) {
	k := &krakendv1.Krakend{
		ObjectMeta: metav1.ObjectMeta{Name: "team1", Namespace: "team1", UID: "uid1"},
		Spec: krakendv1.KrakendSpec{
			Ingress:       krakendv1.Ingress{Hosts: []krakendv1.Host{{Host: "team1.nais.io"}}},
			AuthProviders: []krakendv1.AuthProvider{{Name: "maskinporten", Alg: "RS256", JwkUrl: "https://maskinporten.no/jwk", Issuer: "https://maskinporten.no/"}},
		},
	}
	eps := []krakendv1.ApiEndpoints{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app1", Namespace: "team1",
			OwnerReferences: []metav1.OwnerReference{{UID: "uid1"}},
		},
		Spec: krakendv1.ApiEndpointsSpec{
			AppName: "app1",
			Auth:    krakendv1.Auth{Name: "maskinporten"},
			OpenEndpoints: []krakendv1.Endpoint{{
				Path: "/app1/hello", Method: "GET", BackendHost: "http://app1", BackendPath: "/hello",
			}},
		},
	}}

	objs, err := Convert(k, eps)
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 2 {
		t.Fatalf("expected app and one configmap, got %d objects", len(objs))
	}

	app := objs[0].(*nais_io_v1alpha1.Application)
	if app.Spec.Image != "krakend:"+krakend.ImageTag {
		t.Errorf("unexpected image %q", app.Spec.Image)
	}
	if len(app.Spec.FilesFrom) != 1 || app.Spec.FilesFrom[0].ConfigMap != "team1-gw-config" {
		t.Errorf("unexpected filesFrom %+v", app.Spec.FilesFrom)
	}

	if len(app.Spec.AccessPolicy.Outbound.Rules) != 1 || app.Spec.AccessPolicy.Outbound.Rules[0].Application != "app1" {
		t.Errorf("expected outbound rule to app1 from openEndpoints, got %+v", app.Spec.AccessPolicy.Outbound.Rules)
	}

	cm := objs[1].(*corev1.ConfigMap)
	cfg := cm.Data[krakend.ConfigFileKey]
	if !strings.Contains(cfg, `"version": 4`) || !strings.Contains(cfg, `"endpoint": "/app1/hello"`) {
		t.Errorf("unexpected config:\n%s", cfg)
	}
}
