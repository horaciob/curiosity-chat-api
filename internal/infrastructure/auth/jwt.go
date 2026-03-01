package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService validates JWT tokens and extracts the user ID from the sub claim.
type JWTService struct {
	secret []byte
}

// NewJWTService creates a new JWTService.
func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: []byte(secret)}
}

// Validate parses and validates the token, returning the user ID (sub claim).
func (s *JWTService) Validate(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token claims")
	}

	userID := claims.Subject
	if userID == "" {
		return "", errors.New("token missing subject claim")
	}

	return userID, nil
}
