package main

import (
	"cloud.google.com/go/iam"
	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/nais/krakend/api/v1"
	log "github.com/sirupsen/logrus"
	"strings"
)

type Api struct {
	Ingress          string `json:"ingress,omitempty"`
	Auth             Auth   `json:"auth,omitempty"`
	Team             string `json:"team,omitempty"`
	DocumentationUrl string `json:"documentationUrl,omitempty"`
}

type Auth struct {
	Name  string   `json:"name"`
	Scope []string `json:"scope,omitempty"`
}

type Exporter struct {
	client     *storage.Client
	projectId  string
	location   string
	bucketName string
	bucket     *storage.BucketHandle
}

func NewExporter(ctx context.Context, projectId, location, bucketName string) (*Exporter, error) {
	c, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Exporter{
		client:     c,
		projectId:  projectId,
		location:   location,
		bucketName: bucketName,
		bucket:     c.Bucket(bucketName),
	}, nil
}

func (e *Exporter) Close() error {
	return e.client.Close()
}

func (e *Exporter) UploadDocumentation(ctx context.Context, krakends []*v1.Krakend, endpoints []*v1.ApiEndpoints) error {
	data, err := toApiDocumentation(krakends, endpoints)
	if err != nil {
		return err
	}

	if err = e.createBucketIfNotExists(ctx); err != nil {
		return err
	}

	obj := e.bucket.Object("apis.json")
	w := obj.NewWriter(ctx)
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func (e *Exporter) createBucketIfNotExists(ctx context.Context) error {
	exists, err := e.bucket.Attrs(ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "bucket doesn't exist") {
			return err
		}
	}
	if exists != nil {
		log.Debugf("Bucket %s already exists", e.bucketName)
		return nil
	}

	err = e.bucket.Create(ctx, e.projectId, &storage.BucketAttrs{
		Location: e.location,
		UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
			Enabled: true,
		},
	})
	if err != nil {
		return err
	}
	policy, err := e.bucket.IAM().V3().Policy(ctx)
	if err != nil {
		return fmt.Errorf("Bucket().IAM().V3().Policy: %w", err)
	}
	role := "roles/storage.objectViewer"
	policy.Bindings = append(policy.Bindings, &iampb.Binding{
		Role:    role,
		Members: []string{iam.AllUsers},
	})
	if err := e.bucket.IAM().V3().SetPolicy(ctx, policy); err != nil {
		return fmt.Errorf("Bucket().IAM().SetPolicy: %w", err)
	}
	return nil
}

func toApiDocumentation(krakends []*v1.Krakend, endpoints []*v1.ApiEndpoints) ([]byte, error) {
	apis := make([]*Api, 0)
	for _, krakend := range krakends {
		ingress := fmt.Sprintf("https://%s%s", krakend.Spec.Ingress.Hosts[0].Host, krakend.Spec.Ingress.Hosts[0].Paths[0].Path)

		for _, item := range endpoints {
			if item.Namespace != krakend.Namespace {
				continue
			}

			api := &Api{
				Ingress:          ingress,
				Auth:             Auth{Name: item.Spec.Auth.Name, Scope: item.Spec.Auth.Scope},
				Team:             item.Namespace,
				DocumentationUrl: "TODO",
			}
			apis = append(apis, api)
		}

	}
	return json.Marshal(apis)
}
