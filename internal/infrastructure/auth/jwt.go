package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/tokenstore"
)

const accessTokenTTL = time.Hour

type JWTService struct {
	secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: []byte(secret)}
}

func (s *JWTService) Issue(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

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

func (s *JWTService) IssueRefreshToken(ctx context.Context, userID string, store tokenstore.Store) (string, error) {
	raw, err := tokenstore.GenerateRaw()
	if err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}
	hash := tokenstore.Hash(raw)
	if err := store.Save(ctx, userID, hash, tokenstore.RefreshTokenTTL); err != nil {
		return "", fmt.Errorf("saving refresh token: %w", err)
	}
	return raw, nil
}

func (s *JWTService) ValidateAndRotateRefreshToken(
	ctx context.Context,
	rawToken string,
	store tokenstore.Store,
) (newAccessToken, newRawRefresh string, err error) {
	hash := tokenstore.Hash(rawToken)
	userID, err := store.Get(ctx, hash)
	if err != nil {
		return "", "", fmt.Errorf("invalid or expired refresh token: %w", err)
	}
	if err := store.Delete(ctx, hash); err != nil {
		return "", "", fmt.Errorf("revoking refresh token: %w", err)
	}
	newAccess, err := s.Issue(userID)
	if err != nil {
		return "", "", fmt.Errorf("issuing access token: %w", err)
	}
	newRefresh, err := s.IssueRefreshToken(ctx, userID, store)
	if err != nil {
		return "", "", fmt.Errorf("issuing new refresh token: %w", err)
	}
	return newAccess, newRefresh, nil
}
