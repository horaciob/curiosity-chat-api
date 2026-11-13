package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/internalkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func namedKeys(t *testing.T) *internalkeys.Set {
	t.Helper()
	keys, err := internalkeys.Parse("shared-key", "chat-api:chat-key,poi-consumer:")
	require.NoError(t, err)
	return keys
}

func callInternal(mw func(http.Handler) http.Handler, key string) (int, string) {
	var caller string
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller, _ = internalkeys.CallerFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/internal/x", nil)
	if key != "" {
		req.Header.Set("X-Internal-Key", key)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w.Code, caller
}

func TestInternalAuthenticateKeysRecordsTheCaller(t *testing.T) {
	mw := InternalAuthenticateKeys(namedKeys(t))

	code, caller := callInternal(mw, "chat-key")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "chat-api", caller)

	code, caller = callInternal(mw, "shared-key")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, internalkeys.LegacyCaller, caller)
}

func TestInternalAuthenticateKeysRejectsRevokedAndUnknownKeys(t *testing.T) {
	mw := InternalAuthenticateKeys(namedKeys(t))

	for _, key := range []string{"", "wrong", "poi-consumer"} {
		code, _ := callInternal(mw, key)
		assert.Equal(t, http.StatusUnauthorized, code, key)
	}
}

func TestInternalAuthenticateKeysWithNoKeysRejectsEverything(t *testing.T) {
	code, _ := callInternal(InternalAuthenticateKeys(internalkeys.Single("")), "anything")
	assert.Equal(t, http.StatusUnauthorized, code)
}
