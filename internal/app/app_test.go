package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/config"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
)

func newTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "postgres")
}

func testConfig() *config.Config {
	return &config.Config{
		ServerPort:     "0",
		UserAPIURL:     "http://user-api.test",
		InternalAPIKey: "test-key",
		AllowedOrigins: []string{"https://app.test"},
	}
}

func TestMain(m *testing.M) {
	// The handlers log through the package logger; without this they nil-pointer on first use.
	_ = logger.Init(logger.Config{Level: "error", Format: "json"})
	m.Run()
}

func TestBuildProducesAServerAndAHub(t *testing.T) {
	a := Build(testConfig(), newTestDB(t))

	require.NotNil(t, a.Server, "Build must produce a server")
	require.NotNil(t, a.Server.Handler, "the server must have the router mounted")
	require.NotNil(t, a.Hub, "the websocket hub is part of the service")
	assert.Equal(t, ":0", a.Server.Addr)
}

// Timeouts are the difference between a slow client and an exhausted server, and they are
// easy to drop in a refactor because nothing fails when they are missing.
func TestBuildSetsServerTimeouts(t *testing.T) {
	a := Build(testConfig(), newTestDB(t))

	assert.Equal(t, 15*time.Second, a.Server.ReadTimeout)
	assert.Equal(t, 15*time.Second, a.Server.WriteTimeout)
	assert.Equal(t, 60*time.Second, a.Server.IdleTimeout)
}

func TestBuildMountsTheRoutesTheServiceIsFor(t *testing.T) {
	a := Build(testConfig(), newTestDB(t))

	for _, tc := range []struct {
		name   string
		method string
		path   string
	}{
		{"health probe", http.MethodGet, "/health"},
		{"metrics scrape", http.MethodGet, "/metrics"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			a.Server.Handler.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
			assert.NotEqual(t, http.StatusNotFound, rec.Code, "%s should be mounted", tc.path)
		})
	}
}

// Conversations must not be readable without a token. Asserting it here means the check
// cannot be lost by rewiring, which is the one thing a test of main() would have caught.
func TestBuildKeepsConversationsBehindAuth(t *testing.T) {
	a := Build(testConfig(), newTestDB(t))

	rec := httptest.NewRecorder()
	a.Server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code,
		"an unauthenticated request must not reach the handler")
}

func TestRunServesAndShutsDownOnContextCancel(t *testing.T) {
	a := Build(testConfig(), newTestDB(t))

	// Port 0 lets the OS pick a free one, so the test never collides with a real service.
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()

	// Give the listener a moment to come up before asking it to stop.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err, "a cancelled context is a clean shutdown, not a failure")
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after its context was cancelled")
	}
}

func TestRunReturnsTheErrorWhenItCannotListen(t *testing.T) {
	cfg := testConfig()
	// A port that cannot be bound: the service must report it rather than exit the process.
	cfg.ServerPort = "-1"
	a := Build(cfg, newTestDB(t))

	err := a.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

func TestCloseIsSafeWithoutADatabase(t *testing.T) {
	assert.NoError(t, (&App{}).Close())
}
