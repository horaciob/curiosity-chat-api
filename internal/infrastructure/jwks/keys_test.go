package jwks

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type keyServer struct {
	srv   *httptest.Server
	hits  atomic.Int32
	kid   string
	priv  ed25519.PrivateKey
	extra string
}

func newKeyServer(t *testing.T) *keyServer {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	ks := &keyServer{kid: "kid-1", priv: priv}
	ks.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		ks.hits.Add(1)
		fmt.Fprintf(w, `{"keys":[{"kty":"OKP","crv":"Ed25519","kid":%q,"x":%q}%s]}`,
			ks.kid, base64.RawURLEncoding.EncodeToString(pub), ks.extra)
	}))
	t.Cleanup(ks.srv.Close)
	return ks
}

func (ks *keyServer) sign(t *testing.T, kid string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.RegisteredClaims{Subject: "u1"})
	tok.Header["kid"] = kid
	s, err := tok.SignedString(ks.priv)
	require.NoError(t, err)
	return s
}

func parse(k *Keys, token string) (*jwt.Token, error) {
	return jwt.Parse(token, k.Keyfunc, jwt.WithValidMethods(k.Methods()))
}

func TestNewRequiresASecretOrAURL(t *testing.T) {
	_, err := New("", "")
	assert.Error(t, err)
}

func TestVerifiesEdDSATokensFromTheKeySet(t *testing.T) {
	ks := newKeyServer(t)
	k, err := New("", ks.srv.URL)
	require.NoError(t, err)

	tok, err := parse(k, ks.sign(t, "kid-1"))

	require.NoError(t, err)
	assert.True(t, tok.Valid)
	_, err = parse(k, ks.sign(t, "kid-1"))
	require.NoError(t, err)
	assert.Equal(t, int32(1), ks.hits.Load(), "the key set is cached")
}

func TestUnknownKidRefetchesButIsRateLimited(t *testing.T) {
	ks := newKeyServer(t)
	k, _ := New("", ks.srv.URL)
	require.NoError(t, k.Refresh(context.Background()))

	for i := 0; i < 5; i++ {
		_, err := parse(k, ks.sign(t, "made-up"))
		assert.Error(t, err)
	}
	assert.Equal(t, int32(1), ks.hits.Load(), "made-up kids within minRefetch do not hit user-api")
}

func TestRotatedKeyIsPickedUpAfterTheRefetchWindow(t *testing.T) {
	ks := newKeyServer(t)
	k, _ := New("", ks.srv.URL)
	require.NoError(t, k.Refresh(context.Background()))

	ks.kid = "kid-2"
	k.nextFetch = time.Now().Add(-time.Second)

	_, err := parse(k, ks.sign(t, "kid-2"))
	assert.NoError(t, err)
}

func TestHS256OnlyWhenASecretIsConfigured(t *testing.T) {
	secret := strings.Repeat("s", 32)
	hs, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: "u1"}).SignedString([]byte(secret))
	require.NoError(t, err)
	ks := newKeyServer(t)

	both, _ := New(secret, ks.srv.URL)
	_, err = parse(both, hs)
	assert.NoError(t, err)

	keysOnly, _ := New("", ks.srv.URL)
	_, err = parse(keysOnly, hs)
	assert.Error(t, err)

	secretOnly, _ := New(secret, "")
	_, err = parse(secretOnly, ks.sign(t, "kid-1"))
	assert.Error(t, err, "EdDSA is refused without a JWKS URL")
}

func TestRefusesUnsignedAndKidlessTokens(t *testing.T) {
	ks := newKeyServer(t)
	k, _ := New(strings.Repeat("s", 32), ks.srv.URL)

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{Subject: "u1"}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, err = parse(k, none)
	assert.Error(t, err)

	_, err = parse(k, ks.sign(t, ""))
	assert.Error(t, err)
}

func TestSkipsMalformedKeys(t *testing.T) {
	ks := newKeyServer(t)
	ks.extra = `,{"kty":"RSA","kid":"rsa"},{"kty":"OKP","crv":"Ed25519","kid":"short","x":"AAAA"}`
	k, _ := New("", ks.srv.URL)

	require.NoError(t, k.Refresh(context.Background()))

	assert.Len(t, k.keys, 1)
}

func TestRefreshReportsAnUnreachableKeySet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer srv.Close()
	k, _ := New("", srv.URL)

	assert.Error(t, k.Refresh(context.Background()))
}

func TestAnEmptySecretAcceptsNothing(t *testing.T) {
	k := Secret("")
	forged, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: "u1"}).SignedString([]byte(""))
	require.NoError(t, err)

	_, err = parse(k, forged)

	assert.Error(t, err)
	assert.Equal(t, []string{noMethod}, k.Methods(), "never an empty list, which would disable the algorithm check")
}

func TestAFailedFetchIsRetriedSoon(t *testing.T) {
	old := failedRefetch
	failedRefetch = 50 * time.Millisecond
	t.Cleanup(func() { failedRefetch = old })

	ks := newKeyServer(t)
	up := atomic.Bool{}
	gate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !up.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		ks.srv.Config.Handler.ServeHTTP(w, r)
	}))
	defer gate.Close()
	k, _ := New("", gate.URL)

	assert.Error(t, k.Refresh(context.Background()), "user-api is not up yet")
	up.Store(true)
	time.Sleep(60 * time.Millisecond)

	_, err := parse(k, ks.sign(t, "kid-1"))
	assert.NoError(t, err, "the key set is fetched again shortly after a failure, not after minRefetch")
}
