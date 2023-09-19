package krakend

import (
	"fmt"
	"github.com/luraproject/lura/config"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type Krakend struct {
	log             *logrus.Logger
	k8sClient       *kubernetes.Clientset
	krakendNs       string
	krakendPartials string
	endpointsKey    string
}

func New(log *logrus.Logger, k8sClient *kubernetes.Clientset, krakendNs, krakendPartials, endpointsKey string) *Krakend {
	return &Krakend{
		log:             log,
		k8sClient:       k8sClient,
		krakendNs:       krakendNs,
		krakendPartials: krakendPartials,
		endpointsKey:    endpointsKey,
	}
}

func (k *Krakend) Add(new any) {
	k.log.Debug("received add event")
	endpointsCm := new.(*v1.ConfigMap)

	err := k.ensureEndpoints(endpointsCm)
	if err != nil {
		k.log.Errorf("ensuring endpoints: %v", err)
		return
	}
}

func (k *Krakend) Update(old any, new any) {
	k.log.Debug("received update event")
	oldInstance := old.(*v1.ConfigMap)
	newInstance := new.(*v1.ConfigMap)
	k.log.Infof("old instance: %v", oldInstance.Data)
	k.log.Infof("new instance: %v", newInstance.Data)
	if compareMaps(oldInstance.Data, newInstance.Data) {
		k.log.Infof("old and new instance are the same")
		return
	}

	if oldInstance.GetDeletionTimestamp() != nil {
		k.log.Infof("resource %s in namespace %s is being deleted, ignoring", oldInstance.GetName(), oldInstance.GetNamespace())
		return
	}
	//check diff and update partials
	err := k.ensureEndpoints(newInstance)
	if err != nil {
		k.log.Errorf("ensuring endpoints: %v", err)
		return
	}
}

func (k *Krakend) Delete(new any) {
	k.log.Debug("received delete event")
	//TODO
}

func (k *Krakend) loadKrakendPartials() ([]*config.EndpointConfig, error) {
	k.log.Infof("loading krakend partials from configmap %s in namespace %s", k.krakendPartials, k.krakendNs)
	return nil, nil
}

func (k *Krakend) ensureEndpoints(cm *v1.ConfigMap) error {
	k.log.Infof("endpoints cm: %v", cm.Data)
	if cm.Data[k.endpointsKey] == "" {
		k.log.Infof("endpoints key %s is empty, ignoring", k.endpointsKey)
		return nil
	}

	// TODO: investigate performance and when to load partials
	allEndpoints, err := k.loadKrakendPartials()
	if err != nil {
		return fmt.Errorf("loading krakend partials: %w", err)
	}
	endpoints, err := parseEndpoints([]byte(cm.Data[k.endpointsKey]))
	if err != nil {
		return fmt.Errorf("parsing endpoints: %w", err)
	}
	err = k.mergeEndpoints(allEndpoints, endpoints)
	if err != nil {
		return fmt.Errorf("merging endpoints: %w", err)
	}
	return nil
}

func (k *Krakend) mergeEndpoints(endpoints []*config.EndpointConfig, endpoints2 []*config.EndpointConfig) error {
	return nil
}

func compareMaps(map1, map2 map[string]string) bool {
	if len(map1) != len(map2) {
		return false
	}

	for key, value := range map1 {
		if val, ok := map2[key]; !ok || val != value {
			return false
		}
	}
	return true
}
