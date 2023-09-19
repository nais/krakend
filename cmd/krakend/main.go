package main

import (
	"context"
	"flag"
	"fmt"
	apimachineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"krakend/internal/krakend"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	// Load all client-go auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/client-go/rest"
)

var (
	logLevel        string
	bindAddress     string
	krakendNs       string
	krakendPartials string
	endpointsKey    string
)

func init() {
	flag.StringVar(&bindAddress, "bind-address", ":8080", "Bind address")
	flag.StringVar(&logLevel, "log-level", "debug", "Which log level to output")
	flag.StringVar(&krakendNs, "krakend-ns", "nais-system", "Krakend namespace")
	flag.StringVar(&krakendPartials, "krakend-partials", "cm-partials", "Krakend partials configmap")
	flag.StringVar(&endpointsKey, "endpoints-key", "endpoints.tmpl", "Endpoints key")
}

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	log := newLogger()

	var kubeConfig *rest.Config
	var err error
	if envConfig := os.Getenv("KUBECONFIG"); envConfig != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", envConfig)
		if err != nil {
			panic(err.Error())
		}
		log.Infof("starting endpointer with kubeconfig: %s", envConfig)
	} else {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			log.WithError(err).Fatal("failed to get kubeconfig")
		}
		log.Infof("starting endpointer with in-cluster config: %s", kubeConfig.Host)
	}

	k8sClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.WithError(err).Fatal("setting up k8s client")
	}

	factory := informers.NewSharedInformerFactoryWithOptions(k8sClient, 0, informers.WithTweakListOptions(
		func(options *apimachineryv1.ListOptions) {
			options.LabelSelector = "apiGateway"
			log.Infof("setting label selector: %s", options.LabelSelector)
		}),
	)

	cmInformer := factory.Core().V1().ConfigMaps().Informer()
	if err != nil {
		log.WithError(err).Fatal("setting up cm informer")
	}

	ep := krakend.New(log, k8sClient, krakendNs, krakendPartials, endpointsKey)
	cmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ep.Add,
		UpdateFunc: ep.Update,
		DeleteFunc: ep.Delete,
	})

	go cmInformer.Run(ctx.Done())

	if waitForCacheSync(ctx.Done(), cmInformer.HasSynced) {
		log.Info("cache synced")
	}

	<-ctx.Done()
	log.Info("shutting down")
}

func errorHandler(r *cache.Reflector, err error) {
	fmt.Println("watch error ", err)
}

func newLogger() *logrus.Logger {
	log := logrus.StandardLogger()
	log.SetFormatter(&logrus.JSONFormatter{})

	l, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(l)
	return log
}

func waitForCacheSync(stop <-chan struct{}, cacheSyncs ...cache.InformerSynced) bool {
	max := time.Millisecond * 100
	delay := time.Millisecond
	f := func() bool {
		for _, syncFunc := range cacheSyncs {
			if !syncFunc() {
				return false
			}
		}
		return true
	}
	for {
		select {
		case <-stop:
			return false
		default:
		}
		res := f()
		if res {
			return true
		}
		delay *= 2
		if delay > max {
			delay = max
		}

		select {
		case <-stop:
			return false
		case <-time.After(delay):
		}
	}
}
