// Package jwks resolves the keys that verify user-api access tokens.
//
// user-api signs EdDSA when it has a signing key and publishes the public half at
// /.well-known/jwks.json; while JWT_SECRET is still configured it may also have issued
// HS256 tokens. Keys accepts whichever of the two this service is configured for, and
// refuses everything else (including "none").
//
// The same file lives in poi-api and chat-api; keep the copies identical.
package jwks

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// After a successful fetch, an unknown kid can trigger another one only every minRefetch,
// so a flood of tokens with made-up kids cannot become a flood of requests to user-api.
// After a failed fetch (user-api not up yet) the wait is only failedRefetch: otherwise a
// service that started first would refuse every token for half a minute.
var (
	minRefetch    = 30 * time.Second
	failedRefetch = 2 * time.Second
)

// Keys verifies tokens with a shared secret, a remote JWKS, or both.
type Keys struct {
	secret []byte
	url    string
	client *http.Client

	mu        sync.RWMutex
	keys      map[string]ed25519.PublicKey
	nextFetch time.Time
}

// New builds Keys. Either may be empty, not both.
func New(secret, jwksURL string) (*Keys, error) {
	if secret == "" && jwksURL == "" {
		return nil, errors.New("JWT_JWKS_URL or JWT_SECRET is required")
	}
	return &Keys{
		secret: []byte(secret),
		url:    jwksURL,
		client: &http.Client{Timeout: 5 * time.Second},
		keys:   map[string]ed25519.PublicKey{},
	}, nil
}

// Secret builds Keys for a bare HMAC secret. An empty secret accepts no token at all.
func Secret(secret string) *Keys {
	return &Keys{secret: []byte(secret), client: &http.Client{Timeout: 5 * time.Second}, keys: map[string]ed25519.PublicKey{}}
}

// noMethod is never a real algorithm: jwt.WithValidMethods with an empty list would
// disable the algorithm check, so an unconfigured Keys passes this instead.
const noMethod = "no-method-configured"

// Methods are the algorithms to pass to jwt.WithValidMethods. Never empty.
func (k *Keys) Methods() []string {
	m := []string{noMethod}
	if k.url != "" {
		m = append(m, jwt.SigningMethodEdDSA.Alg())
	}
	if len(k.secret) > 0 {
		m = append(m, jwt.SigningMethodHS256.Alg())
	}
	if len(m) > 1 {
		m = m[1:]
	}
	return m
}

// Keyfunc is a jwt.Keyfunc.
func (k *Keys) Keyfunc(t *jwt.Token) (interface{}, error) {
	switch t.Method.(type) {
	case *jwt.SigningMethodHMAC:
		if len(k.secret) == 0 {
			return nil, errors.New("HS256 tokens are not accepted")
		}
		return k.secret, nil
	case *jwt.SigningMethodEd25519:
		if k.url == "" {
			return nil, errors.New("EdDSA tokens are not accepted")
		}
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token has no kid")
		}
		return k.publicKey(kid)
	}
	return nil, fmt.Errorf("unexpected signing method %s", t.Method.Alg())
}

func (k *Keys) publicKey(kid string) (ed25519.PublicKey, error) {
	k.mu.RLock()
	key, ok := k.keys[kid]
	recent := time.Now().Before(k.nextFetch)
	k.mu.RUnlock()
	if ok {
		return key, nil
	}
	if recent {
		return nil, errors.New("unknown key id")
	}
	if err := k.Refresh(context.Background()); err != nil {
		return nil, err
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	if key, ok := k.keys[kid]; ok {
		return key, nil
	}
	return nil, errors.New("unknown key id")
}

// Refresh fetches the key set now. Call it at startup to fail fast on a bad URL; it is
// also called on an unknown kid (at most every minRefetch).
func (k *Keys) Refresh(ctx context.Context) error {
	err := k.fetch(ctx)
	k.mu.Lock()
	if err != nil {
		k.nextFetch = time.Now().Add(failedRefetch)
	} else {
		k.nextFetch = time.Now().Add(minRefetch)
	}
	k.mu.Unlock()
	return err
}

func (k *Keys) fetch(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.url, nil)
	if err != nil {
		return err
	}
	resp, err := k.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch JWKS: status %d", resp.StatusCode)
	}
	var set struct {
		Keys []struct {
			Kty, Crv, Kid, X string
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}
	keys := map[string]ed25519.PublicKey{}
	for _, jwk := range set.Keys {
		if jwk.Kty != "OKP" || jwk.Crv != "Ed25519" || jwk.Kid == "" {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(jwk.X)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			continue
		}
		keys[jwk.Kid] = ed25519.PublicKey(raw)
	}
	k.mu.Lock()
	k.keys = keys
	k.mu.Unlock()
	return nil
}
