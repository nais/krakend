package main

import (
	"context"
	"fmt"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const AcceptableClockSkew = 5 * time.Second

type JWTAuth struct {
	RequiredClaims map[string]any
	jwksURL        string
	jwksCache      *jwk.Cache
}

func JWT(jwksURL string, requiredClaims map[string]any) (*JWTAuth, error) {
	a := &JWTAuth{
		jwksURL:        jwksURL,
		RequiredClaims: requiredClaims,
	}
	c, err := a.cache()
	if err != nil {
		return nil, err
	}
	a.jwksCache = c
	return a, nil
}

func (p *JWTAuth) ValidateBearerToken(r *http.Request) (jwt.Token, error) {
	header := r.Header.Get("Authorization")

	token := strings.ReplaceAll(header, "Bearer ", "")
	token = strings.TrimSpace(token)

	if token == "" {
		log.Debugf("no JWT token found in request")
		return nil, fmt.Errorf("missing token from header")
	}
	t, err := p.validate(r.Context(), token)
	if err != nil {
		log.Debugf("invalid JWT token: %v", err)
		return nil, fmt.Errorf("invalid token %w", err)
	}
	return t, nil
}

func (p *JWTAuth) validate(ctx context.Context, token string) (jwt.Token, error) {

	opts := []jwt.ValidateOption{
		jwt.WithAcceptableSkew(AcceptableClockSkew),
	}
	for k, v := range p.RequiredClaims {
		switch k {
		case "iss":
			opts = append(opts, jwt.WithIssuer(v.(string)))
		case "aud":
			opts = append(opts, jwt.WithAudience(v.(string)))
		default:
			opts = append(opts, jwt.WithClaimValue(k, v))
		}
	}

	t, err := p.parseToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("parsing jwt: %w", err)
	}
	if err := jwt.Validate(t, opts...); err != nil {
		return nil, fmt.Errorf("validating jwt: %w", err)
	}
	return t, nil
}

func (p *JWTAuth) parseToken(ctx context.Context, raw string) (jwt.Token, error) {
	jwks, err := p.jwksCache.Get(ctx, p.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("provider: fetching jwks: %w", err)
	}
	parseOpts := []jwt.ParseOption{
		jwt.WithKeySet(jwks,
			jws.WithInferAlgorithmFromKey(true),
		),
		jwt.WithAcceptableSkew(AcceptableClockSkew),
	}
	return jwt.ParseString(raw, parseOpts...)
}

func (p *JWTAuth) cache() (*jwk.Cache, error) {
	ctx := context.Background()
	cache := jwk.NewCache(ctx)

	err := cache.Register(p.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("registering jwks provider uri to cache: %w", err)
	}

	// trigger initial fetch and cache of jwk set
	_, err = cache.Refresh(ctx, p.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("initial fetch of jwks from provider: %w", err)
	}
	return cache, nil
}
