package kubernetes

import (
	krakendv1 "github.com/nais/krakend/api/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClient() (client.Client, string) {
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath()},
		&clientcmd.ConfigOverrides{},
	)
	restCfg, err := cc.ClientConfig()
	if err != nil {
		log.Fatalf("kubeconfig: %v", err)
	}
	scheme := runtime.NewScheme()
	if err := krakendv1.AddToScheme(scheme); err != nil {
		log.Fatalf("scheme: %v", err)
	}
	if err = v1.AddToScheme(scheme); err != nil {
		log.Fatalf("scheme: %v", err)
	}
	c, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	ns, _, err := cc.Namespace()
	if err != nil || ns == "" {
		ns = "default"
	}
	return c, ns
}

func kubeconfigPath() string {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env // supports list of paths
	}
	home := homedir.HomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}
