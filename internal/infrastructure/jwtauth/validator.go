// Package jwtauth validates the access tokens user-api issues, locally, with the shared
// JWT_SECRET. chat-api used to ask user-api over HTTP on every WebSocket connect and
// every REST call.
package jwtauth

import (
	"errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/jwks"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// These must match what user-api signs.
const (
	issuer   = "curiosity-user-api"
	audience = "curiosity-api"
	leeway   = 5 * time.Second
)

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// Validator checks app-user access tokens.
type Validator struct {
	keys *jwks.Keys
}

// New creates a Validator that accepts HS256 tokens signed with the shared secret.
func New(secret string) *Validator {
	return NewWithKeys(jwks.Secret(secret))
}

// NewWithKeys creates a Validator for the given key set (user-api's JWKS and/or secret).
func NewWithKeys(keys *jwks.Keys) *Validator {
	return &Validator{keys: keys}
}

// Validate returns the user ID of a valid app-user token. Admin tokens are rejected:
// their subject is an admin account, not someone who can chat.
func (v *Validator) Validate(tokenString string) (string, error) {
	id, _, err := v.ValidateWithExpiry(tokenString)
	return id, err
}

// ValidateWithExpiry is Validate plus when the token stops being valid, so a WebSocket
// opened with it can be closed then instead of living on with an expired credential.
func (v *Validator) ValidateWithExpiry(tokenString string) (string, time.Time, error) {
	var c claims
	token, err := jwt.ParseWithClaims(tokenString, &c, v.keys.Keyfunc,
		jwt.WithValidMethods(v.keys.Methods()),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(leeway),
	)
	if err != nil || !token.Valid {
		return "", time.Time{}, errors.New("invalid or expired token")
	}
	if c.Role != "" {
		return "", time.Time{}, errors.New("admin token used as a user token")
	}
	if c.Subject == "" {
		return "", time.Time{}, errors.New("token has no subject")
	}
	var exp time.Time
	if c.ExpiresAt != nil {
		exp = c.ExpiresAt.Time
	}
	return c.Subject, exp, nil
}
