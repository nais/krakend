package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	log "github.com/sirupsen/logrus"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"time"
)

func main() {
	kubeConfig := setupKubeConfig()
	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	ctx := context.Background()

	endpoint, token := createAssertion(client, ctx, "debugclient-maskinporten", "plattformsikkerhet")

	resp, err := http.DefaultClient.PostForm(endpoint, map[string][]string{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {token},
	})
	if err != nil {
		log.Fatalf("failed to post form: %v", err)
	}

	status := resp.StatusCode
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("failed to read body: %v", err)
	}
	if status != http.StatusOK {
		log.Fatalf("failed to get token, status: %d, body: %s", status, body)
	}

	tokenResp := map[string]any{}
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		log.Fatalf("failed to unmarshal: %v", err)
	}
	fmt.Printf("%s", tokenResp["access_token"])
}

func createAssertion(client *kubernetes.Clientset, ctx context.Context, secretName, namespace string) (string, string) {
	s, err := client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("failed to get secret: %v", err)
	}
	aud := s.Data["MASKINPORTEN_ISSUER"]
	clientId := s.Data["MASKINPORTEN_CLIENT_ID"]
	jwkBytes := s.Data["MASKINPORTEN_CLIENT_JWK"]
	scopes := s.Data["MASKINPORTEN_SCOPES"]
	endpoint := s.Data["MASKINPORTEN_TOKEN_ENDPOINT"]

	key, err := jwk.Parse(jwkBytes)
	if err != nil {
		log.Fatalf("failed to parse jwk: %v", err)
	}

	tok, err := token(map[string]string{
		"aud":   string(aud),
		"iss":   string(clientId),
		"scope": string(scopes),
		"jti":   string(uuid.NewUUID()),
	}, time.Now(), 1*time.Minute)

	if err != nil {
		log.Fatalf("failed to create token: %v", err)
	}
	serialized, err := tok.sign(key)
	if err != nil {
		log.Fatalf("failed to sign token: %v", err)
	}

	return string(endpoint), serialized
}

func setupKubeConfig() *rest.Config {
	var kubeConfig *rest.Config
	var err error

	if envConfig := os.Getenv("KUBECONFIG"); envConfig != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", envConfig)
		if err != nil {
			panic(err.Error())
		}
	} else {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			log.WithError(err).Fatal("failed to get kubeconfig")
		}
	}
	return kubeConfig
}

type Token struct {
	jwt.Token
}

func token(claims map[string]string, iat time.Time, exp time.Duration) (*Token, error) {
	expiry := iat.Add(exp)
	accessToken := jwt.New()
	err := accessToken.Set("iat", iat.Unix())
	if err != nil {
		return nil, err
	}
	err = accessToken.Set("exp", expiry.Unix())
	if err != nil {
		return nil, err
	}
	for k, v := range claims {
		err = accessToken.Set(k, v)
		if err != nil {
			return nil, err
		}
	}
	return &Token{accessToken}, nil
}

func (t *Token) sign(set jwk.Set) (string, error) {
	signer, ok := set.Key(0)
	if !ok {
		return "", fmt.Errorf("could not get signer")
	}
	tok, err := t.Clone()
	if err != nil {
		return "", err
	}
	signedToken, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, signer))
	if err != nil {
		return "", err
	}
	return string(signedToken), nil
}
