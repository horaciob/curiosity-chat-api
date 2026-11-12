package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func etagHandler(body string, status int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

func TestETagIsStableForTheSameBody(t *testing.T) {
	h := ETag(etagHandler(`{"data":[]}`, http.StatusOK))

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/pois", nil))

	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/pois", nil))

	tag := first.Header().Get("ETag")
	require.NotEmpty(t, tag)
	assert.Equal(t, tag, second.Header().Get("ETag"))
	assert.Equal(t, `{"data":[]}`, first.Body.String(), "the body must still be delivered")
}

func TestETagChangesWhenTheBodyChanges(t *testing.T) {
	a := httptest.NewRecorder()
	ETag(etagHandler(`{"total":1}`, http.StatusOK)).
		ServeHTTP(a, httptest.NewRequest(http.MethodGet, "/pois", nil))

	b := httptest.NewRecorder()
	ETag(etagHandler(`{"total":2}`, http.StatusOK)).
		ServeHTTP(b, httptest.NewRequest(http.MethodGet, "/pois", nil))

	assert.NotEqual(t, a.Header().Get("ETag"), b.Header().Get("ETag"))
}

func TestETagReturns304WhenTheClientAlreadyHasIt(t *testing.T) {
	h := ETag(etagHandler(`{"data":[]}`, http.StatusOK))

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/pois", nil))
	tag := first.Header().Get("ETag")

	req := httptest.NewRequest(http.MethodGet, "/pois", nil)
	req.Header.Set("If-None-Match", tag)
	second := httptest.NewRecorder()
	h.ServeHTTP(second, req)

	assert.Equal(t, http.StatusNotModified, second.Code)
	assert.Empty(t, second.Body.String(), "a 304 must carry no body")
}

func TestETagAcceptsListsWeakTagsAndTheWildcard(t *testing.T) {
	h := ETag(etagHandler(`{"data":[]}`, http.StatusOK))
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/pois", nil))
	tag := first.Header().Get("ETag")

	for _, header := range []string{
		tag,
		`"other", ` + tag,
		"W/" + tag,
		"*",
	} {
		req := httptest.NewRequest(http.MethodGet, "/pois", nil)
		req.Header.Set("If-None-Match", header)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotModified, rec.Code, "header %q should match", header)
	}
}

func TestETagIgnoresAStaleTag(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/pois", nil)
	req.Header.Set("If-None-Match", `"something-else"`)
	rec := httptest.NewRecorder()
	ETag(etagHandler(`{"data":[]}`, http.StatusOK)).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"data":[]}`, rec.Body.String())
}

// An unsafe method must never be answered from a cache, and a write must not be swallowed.
func TestETagLeavesUnsafeMethodsAlone(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req := httptest.NewRequest(method, "/pois", nil)
		req.Header.Set("If-None-Match", "*")
		rec := httptest.NewRecorder()
		ETag(etagHandler(`{"created":true}`, http.StatusCreated)).ServeHTTP(rec, req)

		assert.Empty(t, rec.Header().Get("ETag"), "%s must not be tagged", method)
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, `{"created":true}`, rec.Body.String())
	}
}

// Errors are passed through untouched: a cached 500 would be worse than no cache at all.
func TestETagLeavesNonOKResponsesAlone(t *testing.T) {
	for _, status := range []int{
		http.StatusCreated, http.StatusNoContent,
		http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError,
	} {
		rec := httptest.NewRecorder()
		ETag(etagHandler(`{"error":"x"}`, status)).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/pois", nil))

		assert.Empty(t, rec.Header().Get("ETag"), "status %d must not be tagged", status)
		assert.Equal(t, status, rec.Code)
	}
}

func TestETagSetsCacheControlWithoutOverridingTheHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	ETag(etagHandler(`{}`, http.StatusOK)).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/pois", nil))
	assert.Equal(t, "private, must-revalidate", rec.Header().Get("Cache-Control"),
		"responses are per-user, so only the client's own cache may hold them")

	explicit := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(`{}`))
	})
	rec = httptest.NewRecorder()
	ETag(explicit).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/pois", nil))
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"),
		"a handler that opted out must keep its own directive")
}

// A handler that flushes is streaming; the body is already on the wire and must not be
// buffered, re-hashed, or turned into a 304.
func TestETagStepsAsideForAStreamedResponse(t *testing.T) {
	streaming := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("first "))
		w.(http.Flusher).Flush()
		_, _ = w.Write([]byte("second"))
	})

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	req.Header.Set("If-None-Match", "*")
	rec := httptest.NewRecorder()
	ETag(streaming).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "first second", rec.Body.String())
	assert.Empty(t, rec.Header().Get("ETag"))
}
