package krakend

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	v1 "github.com/nais/krakend/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ImageTag must match the tag in krakend/Dockerfile. Bumped by updatecli (hack/updatecli.d/krakend-image.yaml), keep it on its own line.
const ImageTag = "3.0.2"

const (
	// ImageRepository is our mirror of the official image, see krakend/Dockerfile
	ImageRepository = "europe-north1-docker.pkg.dev/nais-io/nais/images/krakend"
	// MinMajorVersion is the lowest KrakenD major version supported by the generated config (version 4)
	MinMajorVersion = 3

	HTTPPort            = 8080
	ServicePort         = 80
	DefaultReplicas     = 2
	DefaultIngressClass = "external-haproxy"
)

// Fullname is the name of all resources for a Krakend instance: <name>-krakend, or <name> if it already contains "krakend".
// Must stay compatible with the old equinixmetal krakend chart, used for existing installations.
func Fullname(k *v1.Krakend) string {
	name := k.Name
	if !strings.Contains(name, "krakend") {
		name += "-krakend"
	}
	if len(name) > 63 {
		name = name[:63]
	}
	return strings.TrimSuffix(name, "-")
}

// ConfigMapName is the name of the ConfigMap holding krakend.json.
// New name compared to the v2 ConfigMaps, so pods running the old config are not affected during rollout.
func ConfigMapName(k *v1.Krakend) string {
	return Fullname(k) + "-v3"
}

// SelectorLabels are immutable on existing Deployments, do not change them.
func SelectorLabels(k *v1.Krakend) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "krakend",
		"app.kubernetes.io/instance": k.Name,
	}
}

func labels(k *v1.Krakend) map[string]string {
	l := SelectorLabels(k)
	l["app.kubernetes.io/managed-by"] = "krakend-operator"
	return l
}

func objectMeta(k *v1.Krakend) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      Fullname(k),
		Namespace: k.Namespace,
		Labels:    labels(k),
	}
}

// Resources returns the resources for a Krakend instance, except the config ConfigMap and netpols.
// Warnings are spec fields that are ignored and should be reported to the user.
func Resources(k *v1.Krakend) ([]client.Object, []string, error) {
	ingress, err := ingress(k)
	if err != nil {
		return nil, nil, err
	}
	deployment, warnings := deployment(k)
	return []client.Object{
		serviceAccount(k),
		service(k),
		deployment,
		ingress,
		serviceMonitor(k),
	}, warnings, nil
}

// image returns the image to use. Overrides are only used for KrakenD 3.x or newer.
func image(k *v1.Krakend) (string, []string) {
	img := k.Spec.Deployment.Image
	def := ImageRepository + ":" + ImageTag
	if img.Registry == "" && img.Repository == "" && img.Tag == "" {
		return def, nil
	}
	if !SupportedImageTag(img.Tag) {
		return def, []string{
			fmt.Sprintf("ignoring spec.deployment.image (tag %q): only KrakenD %d.x or newer is supported, using operator default", img.Tag, MinMajorVersion),
		}
	}
	repository := img.Repository
	if repository == "" {
		repository = ImageRepository
	}
	if img.Registry != "" {
		repository = img.Registry + "/" + repository
	}
	return repository + ":" + img.Tag, nil
}

// SupportedImageTag returns true if tag is a version with major >= MinMajorVersion, e.g. 3, 3.0 or 3.0.2
func SupportedImageTag(tag string) bool {
	major, _, _ := strings.Cut(strings.TrimPrefix(tag, "v"), ".")
	n, err := strconv.Atoi(major)
	return err == nil && n >= MinMajorVersion
}

func deployment(k *v1.Krakend) (*appsv1.Deployment, []string) {
	spec := k.Spec.Deployment
	img, warnings := image(k)

	if spec.DeploymentType != "" && spec.DeploymentType != "deployment" {
		warnings = append(warnings, fmt.Sprintf("ignoring spec.deployment.deploymentType %q: only deployment is supported", spec.DeploymentType))
	}

	replicas := int32(DefaultReplicas)
	if spec.ReplicaCount > 0 && spec.ReplicaCount <= math.MaxInt32 {
		replicas = int32(spec.ReplicaCount)
	}

	pullPolicy := corev1.PullIfNotPresent
	if spec.Image.PullPolicy != "" {
		pullPolicy = corev1.PullPolicy(spec.Image.PullPolicy)
	}

	podLabels := SelectorLabels(k)
	podLabels["logs.nais.io/flow-loki"] = "true"

	name := Fullname(k)
	env := append([]corev1.EnvVar{{Name: "USAGE_DISABLE", Value: "1"}}, spec.ExtraEnvVars...)
	health := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{Path: "/__health", Port: intstr.FromString("http")},
		},
	}

	meta := objectMeta(k)
	// restart pods when the config ConfigMap changes
	meta.Annotations = map[string]string{"reloader.stakater.com/search": "true"}

	return &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: meta,
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: SelectorLabels(k)},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       ptr.To(intstr.FromInt32(1)),
					MaxUnavailable: ptr.To(intstr.FromInt32(0)),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: map[string]string{"kubectl.kubernetes.io/default-container": name},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: name,
					SecurityContext: &corev1.PodSecurityContext{
						SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
					Containers: []corev1.Container{{
						Name:            name,
						Image:           img,
						ImagePullPolicy: pullPolicy,
						Command:         []string{"/usr/bin/krakend"},
						Args:            []string{"run", "-c", "/etc/krakend/" + ConfigFileKey, "-p", strconv.Itoa(HTTPPort)},
						Env:             env,
						Ports: []corev1.ContainerPort{
							{Name: "http", ContainerPort: HTTPPort, Protocol: corev1.ProtocolTCP},
							{Name: "metrics", ContainerPort: MetricsPort, Protocol: corev1.ProtocolTCP},
						},
						LivenessProbe:  health,
						ReadinessProbe: health,
						Resources:      spec.Resources,
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr.To(false),
							RunAsNonRoot:             ptr.To(true),
							RunAsUser:                ptr.To(int64(1000)),
							ReadOnlyRootFilesystem:   ptr.To(true),
							Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "tmp", MountPath: "/tmp"},
							{Name: "config", MountPath: "/etc/krakend", ReadOnly: true},
						},
					}},
					Volumes: []corev1.Volume{
						{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: ConfigMapName(k)},
						}}},
					},
				},
			},
		},
	}, warnings
}

func service(k *v1.Krakend) *corev1.Service {
	return &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: objectMeta(k),
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: SelectorLabels(k),
			Ports: []corev1.ServicePort{
				{Name: "http", Port: ServicePort, TargetPort: intstr.FromInt32(HTTPPort), Protocol: corev1.ProtocolTCP},
				{Name: "metric", Port: MetricsPort, TargetPort: intstr.FromInt32(MetricsPort), Protocol: corev1.ProtocolTCP},
			},
		},
	}
}

func serviceAccount(k *v1.Krakend) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: objectMeta(k),
	}
}

// ingress is always created, spec.ingress.enabled has never been used to disable it
func ingress(k *v1.Krakend) (*networkingv1.Ingress, error) {
	spec := k.Spec.Ingress
	hosts := spec.Hosts
	if len(hosts) == 0 {
		if k.Spec.IngressHost == "" {
			return nil, fmt.Errorf("either ingressHost or ingress.hosts must be specified")
		}
		hosts = []v1.Host{{Host: k.Spec.IngressHost, Paths: []v1.Path{{Path: "/"}}}}
	}

	className := spec.ClassName
	if className == "" {
		className = DefaultIngressClass
	}

	backend := networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: Fullname(k),
			Port: networkingv1.ServiceBackendPort{Number: ServicePort},
		},
	}

	rules := make([]networkingv1.IngressRule, 0, len(hosts))
	for _, h := range hosts {
		paths := make([]networkingv1.HTTPIngressPath, 0, len(h.Paths))
		for _, p := range h.Paths {
			pathType := networkingv1.PathTypePrefix
			if p.PathType != "" {
				pathType = networkingv1.PathType(p.PathType)
			}
			paths = append(paths, networkingv1.HTTPIngressPath{Path: p.Path, PathType: &pathType, Backend: backend})
		}
		rules = append(rules, networkingv1.IngressRule{
			Host:             h.Host,
			IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: paths}},
		})
	}

	meta := objectMeta(k)
	meta.Annotations = spec.Annotations

	return &networkingv1.Ingress{
		TypeMeta:   metav1.TypeMeta{APIVersion: "networking.k8s.io/v1", Kind: "Ingress"},
		ObjectMeta: meta,
		Spec: networkingv1.IngressSpec{
			IngressClassName: &className,
			Rules:            rules,
		},
	}, nil
}

// serviceMonitor is unstructured to avoid depending on prometheus-operator
func serviceMonitor(k *v1.Krakend) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"namespaceSelector": map[string]any{"matchNames": []any{k.Namespace}},
			"selector":          map[string]any{"matchLabels": toAnyMap(SelectorLabels(k))},
			"endpoints": []any{map[string]any{
				"interval":      "10s",
				"scrapeTimeout": "10s",
				"path":          "/metrics",
				"targetPort":    int64(MetricsPort),
			}},
		},
	}}
	u.SetAPIVersion("monitoring.coreos.com/v1")
	u.SetKind("ServiceMonitor")
	u.SetName(Fullname(k))
	u.SetNamespace(k.Namespace)
	u.SetLabels(labels(k))
	return u
}

func toAnyMap(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
