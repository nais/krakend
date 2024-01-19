package helm

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"reflect"
	"strings"
)

type Chart struct {
	chart *chart.Chart
}

func LoadChart(chartFile string) (*Chart, error) {
	if _, err := os.Stat(chartFile); err != nil {
		return nil, fmt.Errorf("loading chart '%s': %w", chartFile, err)
	}

	c, err := loader.Load(chartFile)
	if err != nil {
		return nil, err
	}
	return &Chart{
		chart: c,
	}, nil
}

func (c *Chart) ToUnstructured(releaseName string, releaseNamespace string, values chartutil.Values) ([]*unstructured.Unstructured, error) {
	result, err := c.render(releaseName, releaseNamespace, values)
	if err != nil {
		return nil, fmt.Errorf("rendering chart: %w", err)
	}
	log.Infof("rendered %d resources", len(result))

	resources := make([]*unstructured.Unstructured, 0)
	for key, resource := range result {
		log.Debugf("rendering resource: %s", key)
		if !strings.HasSuffix(key, ".yaml") || resource == "\n" || resource == "" {
			log.Debugf("resource '%s' is not yaml or empty", key)
			continue
		}

		var v any
		if err := yaml.Unmarshal([]byte(resource), &v); err != nil {
			return nil, fmt.Errorf("unmarshalling resource '%s': %w", key, err)
		}
		v = repairMapAny(v)

		obj := &unstructured.Unstructured{
			Object: v.(map[string]interface{}),
		}
		resources = append(resources, obj)
	}
	return resources, err
}

func (c *Chart) render(releaseName string, releaseNamespace string, values chartutil.Values) (map[string]string, error) {
	vals, err := chartutil.ToRenderValues(c.chart, values, chartutil.ReleaseOptions{
		Name:      releaseName,
		Namespace: releaseNamespace,
	}, nil)
	if err != nil {
		return nil, err
	}

	overrideValues(vals, values)
	files, err := engine.Engine{Strict: true}.Render(c.chart, vals)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func overrideValues(target, overrides map[string]any) {
	for key, val := range overrides {
		if reflect.TypeOf(val).Kind() == reflect.Map {
			subMap, ok := target[key].(map[string]any)
			if !ok {
				subMap = make(map[string]any)
				target[key] = subMap
			}
			overrideValues(subMap, val.(map[string]any))
		} else {
			target[key] = val
		}
	}
}

func repairMapAny(v any) any {
	switch t := v.(type) {
	case []any:
		for i, v := range t {
			t[i] = repairMapAny(v)
		}
	case map[any]any:
		nm := make(map[string]any)
		for k, v := range t {
			nm[k.(string)] = repairMapAny(v)
		}
		return nm
	}
	return v
}
