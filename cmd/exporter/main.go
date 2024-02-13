package main

import (
	"cloud.google.com/go/iam"
	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	v1 "github.com/nais/krakend/api/v1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type Config struct {
	ProjectID string `required:"true"`
	Location  string `required:"true"`
	Bucket    string `required:"true"`
}

func main() {
	flag.Parse()
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("failed to process envconfig: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	setupLog(envOrDefault("LOG_LEVEL", "debug"))

	var kubeConfig *rest.Config

	if envConfig := os.Getenv("KUBECONFIG"); envConfig != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", envConfig)
		if err != nil {
			panic(err.Error())
		}
		log.Infof("starting with kubeconfig: %s", envConfig)
	} else {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			log.WithError(err).Fatal("failed to get kubeconfig")
		}
		log.Infof("starting with in-cluster config: %s", kubeConfig.Host)
	}

	lister, err := NewLister(kubeConfig)
	if err != nil {
		log.WithError(err).Fatal("setting up dynamic client")
	}
	endpoints, err := lister.ListApiEndpoints(ctx)
	if err != nil {
		log.Fatalf("%v", err)
	}
	krakends, err := lister.ListKrakends(ctx)
	if err != nil {
		log.Fatalf("%v", err)
	}

	exp, err := NewExporter(ctx, cfg.ProjectID, cfg.Location, cfg.Bucket)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer exp.Close()

	err = exp.UploadDocumentation(ctx, krakends, endpoints)
	if err != nil {
		log.Fatalf("%v", err)
	}
}

type Lister struct {
	aclient dynamic.NamespaceableResourceInterface
	kclient dynamic.NamespaceableResourceInterface
}

func NewLister(kubeConfig *rest.Config) (*Lister, error) {
	l := &Lister{}
	c, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	l.aclient = c.Resource(schema.GroupVersionResource{
		Group:    "krakend.nais.io",
		Version:  "v1",
		Resource: "apiendpoints",
	})

	l.kclient = c.Resource(schema.GroupVersionResource{
		Group:    "krakend.nais.io",
		Version:  "v1",
		Resource: "krakends",
	})

	return l, nil
}

func (l *Lister) ListKrakends(ctx context.Context) ([]*v1.Krakend, error) {
	objs, err := l.kclient.Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	list := make([]*v1.Krakend, 0)
	for _, u := range objs.Items {
		a := &v1.Krakend{}
		err = convert(u, a)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (l *Lister) ListApiEndpoints(ctx context.Context) ([]*v1.ApiEndpoints, error) {
	objs, err := l.aclient.Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	list := make([]*v1.ApiEndpoints, 0)
	for _, u := range objs.Items {
		a := &v1.ApiEndpoints{}
		err = convert(u, a)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

type Api struct {
	Ingress          string `json:"ingress,omitempty"`
	Auth             Auth   `json:"auth,omitempty"`
	Team             string `json:"team,omitempty"`
	DocumentationUrl string `json:"documentationUrl,omitempty"`
}

type Auth struct {
	Name  string   `json:"name"`
	Scope []string `json:"scope,omitempty"`
}

type Exporter struct {
	client     *storage.Client
	projectId  string
	location   string
	bucketName string
	bucket     *storage.BucketHandle
}

func NewExporter(ctx context.Context, projectId, location, bucketName string) (*Exporter, error) {
	c, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Exporter{
		client:     c,
		projectId:  projectId,
		location:   location,
		bucketName: bucketName,
		bucket:     c.Bucket(bucketName),
	}, nil
}

func (e *Exporter) Close() error {
	return e.client.Close()
}

func (e *Exporter) UploadDocumentation(ctx context.Context, krakends []*v1.Krakend, endpoints []*v1.ApiEndpoints) error {
	data, err := toApiDocumentation(krakends, endpoints)
	if err != nil {
		return err
	}

	if err = e.createBucketIfNotExists(ctx); err != nil {
		return err
	}

	obj := e.bucket.Object("apis.json")
	w := obj.NewWriter(ctx)
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func (e *Exporter) createBucketIfNotExists(ctx context.Context) error {
	exists, err := e.bucket.Attrs(ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "bucket doesn't exist") {
			return err
		}
	}
	if exists != nil {
		log.Debugf("Bucket %s already exists", e.bucketName)
		return nil
	}

	err = e.bucket.Create(ctx, e.projectId, &storage.BucketAttrs{
		Location: e.location,
		UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
			Enabled: true,
		},
	})
	if err != nil {
		return err
	}
	policy, err := e.bucket.IAM().V3().Policy(ctx)
	if err != nil {
		return fmt.Errorf("Bucket().IAM().V3().Policy: %w", err)
	}
	role := "roles/storage.objectViewer"
	policy.Bindings = append(policy.Bindings, &iampb.Binding{
		Role:    role,
		Members: []string{iam.AllUsers},
	})
	if err := e.bucket.IAM().V3().SetPolicy(ctx, policy); err != nil {
		return fmt.Errorf("Bucket().IAM().SetPolicy: %w", err)
	}
	return nil
}

func toApiDocumentation(krakends []*v1.Krakend, endpoints []*v1.ApiEndpoints) ([]byte, error) {
	apis := make([]*Api, 0)
	for _, krakend := range krakends {
		ingress := fmt.Sprintf("https://%s%s", krakend.Spec.Ingress.Hosts[0].Host, krakend.Spec.Ingress.Hosts[0].Paths[0].Path)

		for _, item := range endpoints {
			if item.Namespace != krakend.Namespace {
				continue
			}

			api := &Api{
				Ingress:          ingress,
				Auth:             Auth{Name: item.Spec.Auth.Name, Scope: item.Spec.Auth.Scope},
				Team:             item.Namespace,
				DocumentationUrl: "TODO",
			}
			apis = append(apis, api)
		}

	}
	return json.Marshal(apis)
}
func convert(obj unstructured.Unstructured, v interface{}) error {
	return runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, v)
}

func envOrDefault(name, val string) string {
	env := os.Getenv(name)
	if env != "" {
		return env
	}
	return val
}

func setupLog(logLevel string) {
	log.SetFormatter(&log.JSONFormatter{})
	l, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(l)
}
