package krakend

import (
	"fmt"
	"os"
	"testing"
)

func TestParseConfig(t *testing.T) {
	content, err := os.ReadFile("testdata/config.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := parseEndpoints(content)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cfg {
		fmt.Printf("%+v\n", c)
		fmt.Printf("%s/n", c.Endpoint)
		for _, b := range c.Backend {
			fmt.Printf("%+v\n", b)
		}
	}
	fmt.Printf("%+v\n", cfg)

	/*cfg = flexibleconfig.NewTemplateParser(flexibleconfig.Config{
		Parser:    cfg,
		Partials:  os.Getenv(fcPartials),
		Settings:  os.Getenv(fcSettings),
		Path:      os.Getenv(fcPath),
		Templates: os.Getenv(fcTemplates),
	})*/

}
