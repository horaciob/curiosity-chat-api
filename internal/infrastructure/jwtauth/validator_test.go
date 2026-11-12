package jwtauth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "test-secret-long-enough-for-hs256"

func sign(t *testing.T, c jwt.Claims, key string) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(key))
	require.NoError(t, err)
	return s
}

func userClaims(sub string, exp time.Time) jwt.MapClaims {
	return jwt.MapClaims{"sub": sub, "iss": issuer, "aud": []string{audience}, "exp": exp.Unix(), "iat": time.Now().Unix()}
}

func TestValidateAcceptsUserTokens(t *testing.T) {
	id, err := New(secret).Validate(sign(t, userClaims("user-1", time.Now().Add(time.Hour)), secret))
	require.NoError(t, err)
	assert.Equal(t, "user-1", id)
}

func TestValidateRejects(t *testing.T) {
	v := New(secret)
	cases := map[string]string{
		"wrong secret": sign(t, userClaims("u", time.Now().Add(time.Hour)), "another-secret-entirely-xx"),
		"expired":      sign(t, userClaims("u", time.Now().Add(-time.Hour)), secret),
		"wrong issuer": sign(t, jwt.MapClaims{"sub": "u", "iss": "evil", "aud": []string{audience}, "exp": time.Now().Add(time.Hour).Unix()}, secret),
		"no audience":  sign(t, jwt.MapClaims{"sub": "u", "iss": issuer, "exp": time.Now().Add(time.Hour).Unix()}, secret),
		"admin token":  sign(t, jwt.MapClaims{"sub": "a", "role": "admin", "iss": issuer, "aud": []string{audience}, "exp": time.Now().Add(time.Hour).Unix()}, secret),
		"garbage":      "not.a.token",
	}
	for name, tok := range cases {
		_, err := v.Validate(tok)
		assert.Error(t, err, name)
	}
}
