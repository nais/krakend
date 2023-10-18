package utils

import (
	"fmt"
	krakendv1 "github.com/nais/krakend/api/v1"
	log "github.com/sirupsen/logrus"
)

const (
	MsgPathDuplicate = "duplicate paths in apiendpoints resource"
)

func UniquePaths(list *krakendv1.ApiEndpointsList) error {

	paths := make(map[string]string)
	for _, e := range list.Items {
		if e.GetDeletionTimestamp() == nil {
			if len(e.Spec.Endpoints) > 0 {
				for _, p := range e.Spec.Endpoints {
					if _, ok := paths[p.Path]; ok {
						log.Warnf("duplicate path %s in endpoints %s and %s", p.Path, e.Name, paths[p.Path])
						return fmt.Errorf("duplicate path %s in endpoints %s and %s", p.Path, e.Name, paths[p.Path])
					} else {
						paths[p.Path] = e.Name
					}
				}
			}
			if len(e.Spec.OpenEndpoints) > 0 {
				for _, p := range e.Spec.OpenEndpoints {
					if _, ok := paths[p.Path]; ok {
						log.Warnf("duplicate path %s in openEndpoints %s and %s", p.Path, e.Name, paths[p.Path])
						return fmt.Errorf("duplicate path %s in endpoints %s and %s", p.Path, e.Name, paths[p.Path])
					} else {
						paths[p.Path] = e.Name
					}
				}
			}
		}
	}
	return nil
}

func ValidateEndpointsList(el *krakendv1.ApiEndpointsList, e *krakendv1.ApiEndpoints) error {
	endpointUpdated := false
	for i := len(el.Items) - 1; i >= 0; i-- {
		endpoint := el.Items[i]
		// Delete the apiEndpoints that is about to be updated from existing list
		if endpoint.Name == e.Name {
			el.Items = append(el.Items[:i], el.Items[i+1:]...)
			//add new apiEndpoints to list
			el.Items = append(el.Items, *e)
			endpointUpdated = true
		}
	}
	if !endpointUpdated {
		el.Items = append(el.Items, *e)
	}

	err := UniquePaths(el)
	if err != nil {
		return fmt.Errorf(MsgPathDuplicate)
	}
	return nil
}
