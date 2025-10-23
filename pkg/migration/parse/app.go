package parse

import (
	"embed"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"net/url"
	"strings"

	krakendv1 "github.com/nais/krakend/api/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
)

const (
	AppTemplateFile = "templates/app.yaml"
	DefaultImage    = "krakend:2.11.1"
)

//go:embed templates/*.yaml
var templatesDir embed.FS

func Convert(k *krakendv1.Krakend, endpoints ...krakendv1.ApiEndpoints) ([]runtime.Object, error) {
	objs := make([]runtime.Object, 0)
	app := ToApp(k, endpoints...)
	config, err := ToKrakendConfig(k)
	if err != nil {
		return nil, fmt.Errorf("creating krakend config configmap: %v", err)
	}

	filtered := make([]krakendv1.ApiEndpoints, 0)
	for _, e := range endpoints {
		for _, ref := range e.OwnerReferences {
			if ref.UID == k.UID {
				filtered = append(filtered, e)
			}
		}
	}

	partials, err := ToPartialsConfig(k, filtered)
	if err != nil {
		return nil, fmt.Errorf("creating partials config configmap: %v", err)
	}

	objs = append(objs, app, config, partials)
	return objs, nil
}

func ToApp(k *krakendv1.Krakend, endpoints ...krakendv1.ApiEndpoints) *nais_io_v1alpha1.Application {
	app := &nais_io_v1alpha1.Application{}
	err := ParseYaml(templatesDir, AppTemplateFile, app)
	if err != nil {
		log.Fatalf("parsing application template: %v", err)
	}

	app.Name = k.Name
	app.Namespace = k.Namespace
	app.Spec.Image = DefaultImage
	app.Spec.Ingresses = getIngresses(k)
	if k.Spec.Deployment.ReplicaCount > 0 {
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min: &k.Spec.Deployment.ReplicaCount,
		}
	}
	app.Spec.FilesFrom = []nais_io_v1.FilesFrom{
		{
			ConfigMap: fmt.Sprintf("%s-%s-%s", k.Name, "krakend", "config"),
			MountPath: "/etc/krakend",
		},
		{
			ConfigMap: fmt.Sprintf("%s-%s-%s", k.Name, "krakend", "partials"),
			MountPath: "/etc/krakend/partials",
		},
	}

	egressesFromAuth := getEgressesFromAuth(k)
	if len(egressesFromAuth) > 0 {
		app.Spec.AccessPolicy = &nais_io_v1.AccessPolicy{}
		app.Spec.AccessPolicy.Outbound = &nais_io_v1.AccessPolicyOutbound{}
		rules := make([]nais_io_v1.AccessPolicyExternalRule, 0)
		for _, e := range egressesFromAuth {
			rules = append(rules, nais_io_v1.AccessPolicyExternalRule{
				Host: e.ExternalHost,
			})
		}
		app.Spec.AccessPolicy.Outbound.External = rules
	}

	egresses := getEgresses(endpoints...)
	if len(egresses) > 0 {
		if app.Spec.AccessPolicy == nil {
			app.Spec.AccessPolicy = &nais_io_v1.AccessPolicy{}
		}
		if app.Spec.AccessPolicy.Outbound == nil {
			app.Spec.AccessPolicy.Outbound = &nais_io_v1.AccessPolicyOutbound{}
		}

		for _, e := range egresses {
			if e.App != "" {
				if app.Spec.AccessPolicy.Outbound.Rules == nil {
					app.Spec.AccessPolicy.Outbound.Rules = make([]nais_io_v1.AccessPolicyRule, 0)
				}
				app.Spec.AccessPolicy.Outbound.Rules = append(app.Spec.AccessPolicy.Outbound.Rules, nais_io_v1.AccessPolicyRule{
					Application: e.App,
				})
			}
			if e.ExternalHost != "" {
				app.Spec.AccessPolicy.Outbound.External = append(app.Spec.AccessPolicy.Outbound.External, nais_io_v1.AccessPolicyExternalRule{
					Host: e.ExternalHost,
				})
			}
		}
	}

	return app
}

type Egress struct {
	App          string
	ExternalHost string
}

func getEgressesFromAuth(k *krakendv1.Krakend) []*Egress {
	egresses := make([]*Egress, 0)
	for _, a := range k.Spec.AuthProviders {
		u, err := url.Parse(a.JwkUrl)
		if err != nil {
			continue
		}

		egresses = append(egresses, &Egress{
			ExternalHost: u.Host,
		})
	}
	return egresses
}

func getEgresses(endpoints ...krakendv1.ApiEndpoints) []*Egress {
	seen := make(map[string]bool)
	egresses := make([]*Egress, 0)
	for _, ep := range endpoints {
		for _, e := range ep.Spec.Endpoints {
			u, err := url.Parse(e.BackendHost)
			if err != nil {
				log.Warnf("failed to parse backend host %s in ApiEndpoints %s, skipping: %v", e.BackendHost, ep.Name, err)
				continue
			}
			// only support http for service discovery
			if u.Scheme == "http" && u.Hostname() != "" {
				parts := strings.Split(u.Hostname(), ".")
				app := ""
				if len(parts) > 0 {
					app = parts[0]
				}

				if _, ok := seen[app]; !ok && app != "" {
					seen[app] = true
					egresses = append(egresses, &Egress{App: app})
				}
				continue
			}
			if u.Hostname() != "" {
				if _, ok := seen[u.Hostname()]; !ok {
					seen[u.Hostname()] = true
					egresses = append(egresses, &Egress{ExternalHost: u.Hostname()})
				}
			}
		}
	}
	return egresses
}

func getIngresses(k *krakendv1.Krakend) []nais_io_v1.Ingress {
	if len(k.Spec.Ingress.Hosts) == 0 {
		return nil
	}
	ings := make([]nais_io_v1.Ingress, 0)
	for _, host := range k.Spec.Ingress.Hosts {
		ing := "https://" + host.Host
		for _, path := range host.Paths {
			if path.Path == "/" {
				continue
			}
			ing += path.Path
		}
		ings = append(ings, nais_io_v1.Ingress(ing))
	}
	return ings
}
