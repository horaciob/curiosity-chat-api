package tokenstore

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// GenerateRaw produces a 32-byte cryptographically secure random token
// encoded as base64url (no padding). This is the value returned to the client.
func GenerateRaw() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Hash returns the SHA-256 hex digest of the raw token.
// Only the hash is stored in Redis — never the raw token itself.
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
