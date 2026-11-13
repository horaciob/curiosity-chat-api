// Package internalkeys holds the X-Internal-Key values a service accepts, one per caller.
//
// With a single shared key, a leak from any service lets the holder call every internal
// route everywhere, and revoking it means rotating it in every service at once. Named
// keys (INTERNAL_API_KEYS="chat-api:k1,poi-consumer:k2") can be revoked one caller at a
// time, and the caller's name is known for logs and audit.
//
// The same file lives in each Go service; keep the copies identical.
package internalkeys

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
)

// LegacyCaller names the single shared INTERNAL_API_KEY.
const LegacyCaller = "shared"

// Set is the accepted keys by caller name. The zero value accepts nothing.
type Set struct {
	keys []entry
}

type entry struct {
	caller string
	key    []byte
}

// Parse builds a Set from the legacy shared key (may be empty) and a comma-separated list
// of name:key pairs (may be empty). Entries with an empty key are skipped, so
// "chat-api:${UNSET}" disables that caller instead of accepting an empty header.
func Parse(legacyKey, named string) (*Set, error) {
	s := &Set{}
	if legacyKey != "" {
		s.keys = append(s.keys, entry{caller: LegacyCaller, key: []byte(legacyKey)})
	}
	for _, pair := range strings.Split(named, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		caller, key, ok := strings.Cut(pair, ":")
		caller, key = strings.TrimSpace(caller), strings.TrimSpace(key)
		if !ok || caller == "" {
			return nil, fmt.Errorf("INTERNAL_API_KEYS: want name:key, got %q", redact(pair))
		}
		if key == "" {
			continue
		}
		s.keys = append(s.keys, entry{caller: caller, key: []byte(key)})
	}
	return s, nil
}

// Single is a Set with only the legacy shared key.
func Single(key string) *Set {
	s, _ := Parse(key, "")
	return s
}

// Empty reports whether no key is accepted.
func (s *Set) Empty() bool { return s == nil || len(s.keys) == 0 }

// Callers lists the configured caller names, for startup logs (never the keys).
func (s *Set) Callers() []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s.keys))
	for i, e := range s.keys {
		out[i] = e.caller
	}
	return out
}

// Match returns the caller a provided key belongs to. Every configured key is compared in
// constant time, so the response time does not reveal which (or how much of a) key matched.
func (s *Set) Match(provided string) (string, bool) {
	if s == nil || provided == "" {
		return "", false
	}
	caller, found := "", 0
	for _, e := range s.keys {
		if subtle.ConstantTimeCompare([]byte(provided), e.key) == 1 {
			if found == 0 {
				caller = e.caller
			}
			found = 1
		}
	}
	return caller, found == 1
}

type ctxKey struct{}

// WithCaller records the authenticated internal caller.
func WithCaller(ctx context.Context, caller string) context.Context {
	return context.WithValue(ctx, ctxKey{}, caller)
}

// CallerFromContext returns the internal caller recorded by WithCaller.
func CallerFromContext(ctx context.Context) (string, bool) {
	c, ok := ctx.Value(ctxKey{}).(string)
	return c, ok && c != ""
}

func redact(pair string) string {
	if name, _, ok := strings.Cut(pair, ":"); ok {
		return name + ":***"
	}
	if len(pair) > 4 {
		return pair[:4] + "***"
	}
	return "***"
}
