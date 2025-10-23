package parse

import (
	"bytes"
	"embed"
	"fmt"
	"strings"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	k8s_json "k8s.io/apimachinery/pkg/runtime/serializer/json"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func ParseYaml(dir embed.FS, name string, v any) error {
	reader, err := dir.Open(name)
	if err != nil {
		return fmt.Errorf("opening %s: %v", name, err)
	}
	d := yamlutil.NewYAMLOrJSONDecoder(reader, 4096)
	return d.Decode(v)
}

func ToYAML(objs ...runtime.Object) (string, error) {
	out := ""
	for _, obj := range objs {
		s, err := serializeToYAML(obj)
		if err != nil {
			return "", err
		}
		if out != "" {
			out = out + "\n---\n"
		}
		out = out + s
	}
	return out, nil
}

func serializeToYAML(obj runtime.Object) (string, error) {
	// TODO: move to init function
	err := nais_io_v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", err
	}

	s := k8s_json.NewSerializerWithOptions(
		k8s_json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme,
		k8s_json.SerializerOptions{Yaml: true, Pretty: true, Strict: false},
	)
	var buf bytes.Buffer
	if err := s.Encode(obj, &buf); err != nil {
		return "", err
	}
	ret := buf.String()
	// simple hack to remove null creationTimestamp
	ret = strings.ReplaceAll(ret, "  creationTimestamp: null\n", "")
	ret = strings.ReplaceAll(ret, "status: {}\n", "")
	return ret, nil
}
