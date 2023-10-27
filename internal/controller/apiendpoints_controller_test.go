package controller

import (
	krakendv1 "github.com/nais/krakend/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestAppsInNamespace(t *testing.T) {

	r := &ApiEndpointsReconciler{
		ClusterDomain: "cluster.local",
	}

	tt := []struct {
		name       string
		url        string
		assertions func(apps []string)
	}{
		{
			name: "valid - full svc url",
			url:  "http://app1.ns1.svc.cluster.local",
			assertions: func(apps []string) {
				assert.Equal(t, 1, len(apps))
				assert.Equal(t, "app1", apps[0])
			},
		},
		{
			name: "valid - svc url without clusterdomain",
			url:  "http://app1.ns1",
			assertions: func(apps []string) {
				assert.Equal(t, 1, len(apps))
				assert.Equal(t, "app1", apps[0])
			},
		},
		{
			name: "valid - url with only service name",
			url:  "http://app1",
			assertions: func(apps []string) {
				assert.Equal(t, 1, len(apps))
				assert.Equal(t, "app1", apps[0])
			},
		},
		{
			name: "valid - ingress url",
			url:  "https://app1.nais.io",
			assertions: func(apps []string) {
				assert.Equal(t, 0, len(apps))
			},
		},
		{
			name: "invalid - url with different domain",
			url:  "http://app1.ns1.whatever",
			assertions: func(apps []string) {
				assert.Equal(t, 0, len(apps))
			},
		},
	}

	for _, tc := range tt {
		e := &krakendv1.ApiEndpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns1",
			},
			Spec: krakendv1.ApiEndpointsSpec{
				Endpoints: []krakendv1.Endpoint{
					{
						BackendHost: tc.url,
					},
				},
			},
		}
		apps := r.appsInNamespace(e)
		tc.assertions(apps)
	}
}
