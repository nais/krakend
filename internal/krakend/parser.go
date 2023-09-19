package krakend

import (
	"encoding/json"
	"github.com/luraproject/lura/config"
)

func parseEndpoints(content []byte) ([]*config.EndpointConfig, error) {
	endpoints := make([]*config.EndpointConfig, 0)
	err := json.Unmarshal(content, &endpoints)
	if err != nil {
		return nil, err
	}
	return endpoints, nil
}
