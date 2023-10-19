package main

import (
	"encoding/json"
	"flag"
	"github.com/brianvoe/gofakeit/v6"
	krakendv1 "github.com/nais/krakend/api/v1"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

func main() {
	flag.Parse()
	var samplesDir string
	flag.StringVar(&samplesDir, "dir", "config/samples", "dir to write to")
	a := &krakendv1.ApiEndpointsSpec{}
	err := fakeAndSave(filepath.Join(samplesDir, "apiendpoints.json"), a)
	if err != nil {
		log.Fatalf("fakeAndSave: %s", err)
	}
	k := &krakendv1.KrakendSpec{}
	err = fakeAndSave(filepath.Join(samplesDir, "krakend.json"), k)
	if err != nil {
		log.Fatalf("fakeAndSave: %s", err)
	}
}

func fakeAndSave(file string, v any) error {
	err := gofakeit.Struct(v)
	if err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(file, b, 0644)
}
