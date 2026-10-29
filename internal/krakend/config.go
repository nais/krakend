package krakend

import (
	"encoding/json"
	"sort"

	v1 "github.com/nais/krakend/api/v1"
)

const (
	ConfigSchema  = "https://www.krakend.io/schema/v3.0/krakend.json"
	ConfigVersion = 4
	ConfigFileKey = "krakend.json"
	MetricsPort   = 9091
)

type Config struct {
	Schema      string         `json:"$schema"`
	Version     int            `json:"version"`
	Timeout     string         `json:"timeout"`
	CacheTTL    string         `json:"cache_ttl"`
	ExtraConfig map[string]any `json:"extra_config"`
	Endpoints   []*Endpoint    `json:"endpoints"`
}

// BuildConfig renders the complete krakend.json for a Krakend instance and its ApiEndpoints.
func BuildConfig(k *v1.Krakend, list []v1.ApiEndpoints) ([]byte, error) {
	items := make([]v1.ApiEndpoints, len(list))
	copy(items, list)
	// stable order to avoid config changes (and pod restarts) caused by list ordering
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	endpoints, err := ToKrakendEndpoints(k, items)
	if err != nil {
		return nil, err
	}

	cfg := Config{
		Schema:      ConfigSchema,
		Version:     ConfigVersion,
		Timeout:     "3s",
		CacheTTL:    "3s",
		ExtraConfig: serviceExtraConfig(),
		Endpoints:   endpoints,
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func serviceExtraConfig() map[string]any {
	return map[string]any{
		"router": map[string]any{
			"logger_skip_paths": []string{"/__health"},
		},
		"telemetry/opentelemetry": map[string]any{
			"service_name":            "krakend_prometheus_service",
			"metric_reporting_period": 1,
			"exporters": map[string]any{
				"prometheus": []map[string]any{
					{
						"name":            "krakend_app",
						"port":            MetricsPort,
						"process_metrics": true,
						"go_metrics":      true,
					},
				},
			},
			"layers": map[string]any{
				"global": map[string]any{
					"disable_metrics":     false,
					"disable_traces":      false,
					"disable_propagation": false,
				},
				"proxy": map[string]any{
					"disable_metrics": false,
					"disable_traces":  false,
				},
				"backend": map[string]any{
					"metrics": map[string]any{
						"disable_stage":       false,
						"round_trip":          true,
						"read_payload":        true,
						"detailed_connection": true,
					},
					"traces": map[string]any{
						"disable_stage":       false,
						"round_trip":          true,
						"read_payload":        true,
						"detailed_connection": true,
					},
				},
			},
		},
	}
}
