package main

import (
	"context"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	bindAddr string
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func init() {
	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.Parse()
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGINT)
	jwksUrl := envOrDefault("JWKS_URL", "https://test.maskinporten.no/jwk")
	issuer := envOrDefault("ISSUER", "https://test.maskinporten.no/")
	scope := envOrDefault("SCOPE", "")

	requiredClaims := map[string]any{
		"iss": issuer,
	}
	if scope != "" {
		requiredClaims["scope"] = scope
	}

	jwtAuth, err := JWT(jwksUrl, requiredClaims)
	if err != nil {
		log.Fatalf("creating jwt auth configuration: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		token, err := jwtAuth.ValidateBearerToken(r)
		if err != nil {
			fmt.Printf("invalid token: %v", err)
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		scope, _ := token.Get("scope")

		fmt.Printf("got request with valid token for issuer %s and scope %v", token.Issuer(), scope)

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "ok for issuer %s and scope %v", token.Issuer(), scope)
	})

	fmt.Println("running @", bindAddr)
	go func() {
		log.Fatal((&http.Server{Addr: bindAddr}).ListenAndServe())
	}()

	<-interrupt

	fmt.Println("shutting down")

	(&http.Server{Addr: bindAddr}).Shutdown(context.Background())
}
