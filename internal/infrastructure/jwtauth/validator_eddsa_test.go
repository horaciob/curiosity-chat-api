package jwtauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/jwks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jwksServer(t *testing.T) (ed25519.PrivateKey, *jwks.Keys) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"keys":[{"kty":"OKP","crv":"Ed25519","kid":"k1","alg":"EdDSA","x":%q}]}`,
			base64.RawURLEncoding.EncodeToString(pub))
	}))
	t.Cleanup(srv.Close)
	keys, err := jwks.New("", srv.URL)
	require.NoError(t, err)
	return priv, keys
}

func eddsaToken(t *testing.T, priv ed25519.PrivateKey, subject, role string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub": subject, "iss": "curiosity-user-api", "aud": "curiosity-api",
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	}
	if role != "" {
		claims["role"] = role
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tok.Header["kid"] = "k1"
	s, err := tok.SignedString(priv)
	require.NoError(t, err)
	return s
}

func TestValidatorAcceptsEdDSATokensFromUserAPI(t *testing.T) {
	priv, keys := jwksServer(t)

	got, err := NewWithKeys(keys).Validate(eddsaToken(t, priv, "user-1", ""))

	require.NoError(t, err)
	assert.Equal(t, "user-1", got)
}

func TestValidatorRejectsEdDSAAdminTokens(t *testing.T) {
	priv, keys := jwksServer(t)

	_, err := NewWithKeys(keys).Validate(eddsaToken(t, priv, "admin-1", "admin"))

	assert.Error(t, err)
}

func TestEmptySecretValidatorAcceptsNothing(t *testing.T) {
	forged, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "u", "iss": "curiosity-user-api", "aud": "curiosity-api", "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(""))
	require.NoError(t, err)

	_, err = New("").Validate(forged)

	assert.Error(t, err)
}
