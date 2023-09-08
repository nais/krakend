package endpointer

import (
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type Endpointer struct {
	log       *logrus.Logger
	k8sClient *kubernetes.Clientset
}

type AudienceInfo struct {
	InstanceUID  types.UID
	InstanceName string
	Namespace    string
	IP           string
	Owner        string
}

func (e *Endpointer) Add(new any) {
	e.log.Debug("received add event")
	//check labels and add to partials
}

func (e *Endpointer) Update(old any, new any) {
	e.log.Debug("received update event")
	oldInstance := old.(*v1.ConfigMap)
	newInstance := new.(*v1.ConfigMap)
	e.log.Infof("old instance: %v", oldInstance.Data)
	e.log.Infof("new instance: %v", newInstance.Data)
	if CompareMaps(oldInstance.Data, newInstance.Data) {
		e.log.Infof("old and new instance are the same")
		return
	} else {
		e.log.Infof("old and new instance are different")
	}
	if oldInstance.GetDeletionTimestamp() != nil {
		e.log.Infof("resource %s in namespace %s is being deleted, ignoring", oldInstance.GetName(), oldInstance.GetNamespace())
		return
	}
	//check diff and update partials
}

func (e *Endpointer) Delete(new any) {
	e.log.Debug("received delete event")
	//delete from partials
}

func New(log *logrus.Logger, k8sClient *kubernetes.Clientset) *Endpointer {
	return &Endpointer{
		log:       log,
		k8sClient: k8sClient,
	}
}

func CompareMaps(map1, map2 map[string]string) bool {
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
