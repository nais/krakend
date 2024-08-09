package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	_ "github.com/santhosh-tekuri/jsonschema/v5/httploader"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	krakendv1 "github.com/nais/krakend/api/v1"
)

//+kubebuilder:webhook:path=/validate-krakends,mutating=false,timeoutSeconds=30,failurePolicy=fail,sideEffects=None,groups=krakend.nais.io,resources=krakends,verbs=create;update,versions=v1,name=krakends.krakend.nais.io,admissionReviewVersions=v1

const ServiceExtraConfigSchemaUrl = "https://www.krakend.io/schema/v%s/service_extra_config.json"

type KrakendsValidator struct {
	client  client.Client
	config  *rest.Config
	decoder *admission.Decoder
}

func (v *KrakendsValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	v.decoder = admission.NewDecoder(mgr.GetScheme())
	v.client = mgr.GetClient()
	v.config = mgr.GetConfig()
	mgr.GetWebhookServer().Register("/validate-krakends", &webhook.Admission{Handler: v})
	return nil
}

func (v *KrakendsValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	k := &krakendv1.Krakend{}
	err := v.decoder.Decode(req, k)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if err := v.validate(ctx, k); err != nil {
		return admission.Denied(err.Error())
	}
	return admission.Allowed("")
}

func (v *KrakendsValidator) validate(ctx context.Context, k *krakendv1.Krakend) error {
	if k.Spec.Deployment.ExtraConfig != nil {
		var serviceExtraConfig interface{}
		if err := json.Unmarshal(k.Spec.Deployment.ExtraConfig.Raw, &serviceExtraConfig); err != nil {
			return fmt.Errorf("unmarshaling serviceExtraConfig: %w", err)
		}

		var url string
		if strings.Contains(ServiceExtraConfigSchemaUrl, "v%s") {
			version, err := getVersion(ctx, v.config, k)
			if err != nil {
				return fmt.Errorf("getting version: %v", err)
			}

			url = fmt.Sprintf(ServiceExtraConfigSchemaUrl, getMinorVersion(*version))
		} else {
			// the global schema url var might have been set at build time
			// to something different
			url = ServiceExtraConfigSchemaUrl
		}

		sch, err := jsonschema.Compile(url)
		if err != nil {
			return fmt.Errorf("compling serviceExtraConfig json schema: %w", err)
		}

		if err = sch.Validate(serviceExtraConfig); err != nil {
			return fmt.Errorf("linting the serviceExtraConfig: %w", err)
		}
	}

	return nil
}

func getMinorVersion(v string) string {
	comps := strings.Split(v, ".")
	if len(comps) < 2 {
		return v
	}
	return fmt.Sprintf("%s.%s", comps[0], comps[1])
}

func getVersion(ctx context.Context, cfg *rest.Config, k *krakendv1.Krakend) (*string, error) {
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating clientset: %v", err)
	}

	job := &batchv1.Job{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("krakend-detect-version-%s", strconv.FormatInt(time.Now().UTC().Unix(), 10)),
			Namespace: k.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "krakend-detect-version",
							Image:   fmt.Sprintf("%s:%s", k.Spec.Deployment.Image.Repository, k.Spec.Deployment.Image.Tag),
							Command: []string{"krakend"},
							Args:    []string{"version"},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("10Mi"),
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	jobsClient := clientset.BatchV1().Jobs(job.Namespace)
	job, err = jobsClient.Create(ctx, job, v1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating getversion job: %v", err)
	}

	err = wait.PollUntilContextTimeout(ctx, 10*time.Second, 300*time.Second, true, func(ctx context.Context) (bool, error) {
		job, err := jobsClient.Get(ctx, job.Name, v1.GetOptions{})
		if err != nil {
			return false, err
		}

		if job.Status.Succeeded > 0 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("completing job: %v", err)
	}

	podsClient := clientset.CoreV1().Pods(job.Namespace)
	podList, err := podsClient.List(ctx, v1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %v", err)
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pods found for job")
	}

	podName := podList.Items[0].Name

	req := podsClient.GetLogs(podName, &corev1.PodLogOptions{})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting pod logs: %v", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, fmt.Errorf("copy pod logs to buffer: %v", err)
	}

	logs := buf.String()

	re := regexp.MustCompile(`KrakenD Version: (\d+\.\d+\.\d+)`)

	matches := re.FindStringSubmatch(logs)

	if len(matches) < 1 {
		return nil, fmt.Errorf("version not found")
	}

	deletePolicy := v1.DeletePropagationBackground
	if err = jobsClient.Delete(ctx, job.Name, v1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return nil, fmt.Errorf("creating getversion job: %v", err)
	}

	return &matches[1], nil
}
