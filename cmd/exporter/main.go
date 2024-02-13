package main

import (
	"context"
	"flag"
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
	"syscall"
)

type Config struct {
	ProjectID string
	Location  string
	Bucket    string
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

	exporter, err := NewExporter(ctx, cfg.ProjectID, cfg.Location, cfg.Bucket)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer exporter.Close()

	err = exporter.UploadDocumentation(ctx, krakends, endpoints)
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
