package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// ETag makes a repeated read of unchanged data free.
//
// The API sent no validators at all, so a traveller reopening the map paid for the same
// places again on every launch — on roaming data, over a connection that is the scarce
// resource in this app. With an ETag the client asks "still this?" and gets 304 and an empty
// body when nothing has changed.
//
// The tag is the hash of the body actually sent, which is correct by construction: it cannot
// drift from the content the way a hand-maintained version counter does. The cost is that the
// response is still generated; what is saved is the transfer, which is the expensive half on
// a mobile network.
//
// Only safe methods are handled. A POST or DELETE must never be answered from a cache.
func ETag(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		rec := &etagRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		// Only a plain 200 is safe to validate. Errors and redirects are passed through as
		// they came, and a streamed or already-flushed response cannot be rewritten.
		if rec.status != http.StatusOK || rec.flushed {
			rec.flushBody()
			return
		}

		sum := sha256.Sum256(rec.body.Bytes())
		tag := `"` + hex.EncodeToString(sum[:16]) + `"`
		w.Header().Set("ETag", tag)

		// Private: responses are per-user, so only the client's own cache may hold them.
		// must-revalidate keeps the client asking rather than serving stale content silently.
		if w.Header().Get("Cache-Control") == "" {
			w.Header().Set("Cache-Control", "private, must-revalidate")
		}

		if matchesETag(r.Header.Get("If-None-Match"), tag) {
			// A 304 carries no body, and Content-Length must not describe one.
			w.Header().Del("Content-Length")
			w.WriteHeader(http.StatusNotModified)
			return
		}

		rec.flushBody()
	})
}

// matchesETag reports whether the If-None-Match header covers this tag.
//
// Handles the comma-separated list form and the "*" wildcard, and compares weak and strong
// tags by their opaque part, which is what the weak comparison this header calls for means.
func matchesETag(header, tag string) bool {
	if header == "" {
		return false
	}
	if strings.TrimSpace(header) == "*" {
		return true
	}
	for _, candidate := range strings.Split(header, ",") {
		if strings.TrimPrefix(strings.TrimSpace(candidate), "W/") == tag {
			return true
		}
	}
	return false
}

// etagRecorder buffers the body so it can be hashed before anything reaches the client.
type etagRecorder struct {
	http.ResponseWriter
	body    bytes.Buffer
	status  int
	written bool
	flushed bool
}

func (e *etagRecorder) WriteHeader(status int) {
	if e.written {
		return
	}
	e.status = status
	e.written = true
}

func (e *etagRecorder) Write(p []byte) (int, error) {
	if !e.written {
		e.written = true
	}
	// Once the response has started streaming there is nothing left to hash, and buffering
	// the rest would silently truncate it.
	if e.flushed {
		return e.ResponseWriter.Write(p)
	}
	return e.body.Write(p)
}

// Flush marks the response as streamed. Anything already sent cannot be hashed or turned into
// a 304, so the recorder steps aside for the rest of the response.
func (e *etagRecorder) Flush() {
	if !e.flushed {
		e.flushed = true
		e.ResponseWriter.WriteHeader(e.status)
		_, _ = e.ResponseWriter.Write(e.body.Bytes())
		e.body.Reset()
	}
	if f, ok := e.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// flushBody sends the buffered response on to the client.
func (e *etagRecorder) flushBody() {
	if e.flushed {
		return
	}
	e.flushed = true
	e.ResponseWriter.WriteHeader(e.status)
	_, _ = e.ResponseWriter.Write(e.body.Bytes())
}
