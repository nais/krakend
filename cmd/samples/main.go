package main

import (
	"encoding/json"
	"fmt"
	"github.com/brianvoe/gofakeit/v6"
	krakendv1 "github.com/nais/krakend/api/v1"
)

func main() {
	spec := &krakendv1.ApiEndpointsSpec{}
	gofakeit.Struct(spec)
	b, err := json.Marshal(spec)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", b)
}
