package ws

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// replica starts a hub that shares mr with every other replica.
func replica(t *testing.T, mr *miniredis.Miniredis) *Hub {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	hub := NewHub().WithCluster(NewRedisClusterWithClient(rdb))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	before := mr.PubSubNumSub(deliverChannel)[deliverChannel]
	go hub.Run()
	hub.StartCluster(ctx, nil)
	// This replica's subscription is live once miniredis counts one more subscriber.
	require.Eventually(t, func() bool { return mr.PubSubNumSub(deliverChannel)[deliverChannel] > before }, time.Second, 5*time.Millisecond)
	return hub
}

func connect(hub *Hub, userID string) *Client {
	c := &Client{hub: hub, UserID: userID, Conn: &websocket.Conn{}, Send: make(chan []byte, 8)}
	hub.Register(c)
	return c
}

func receive(t *testing.T, c *Client) string {
	t.Helper()
	select {
	case msg := <-c.Send:
		return string(msg)
	case <-time.After(time.Second):
		t.Fatal("no message delivered")
		return ""
	}
}

func TestClusterDeliversToAUserConnectedToAnotherReplica(t *testing.T) {
	mr := miniredis.RunT(t)
	a, b := replica(t, mr), replica(t, mr)
	bob := connect(b, "bob")
	require.Eventually(t, func() bool { return b.IsOnline("bob") }, time.Second, 5*time.Millisecond)

	a.BroadcastJSON("bob", map[string]string{"type": "typing"})

	assert.JSONEq(t, `{"type":"typing"}`, receive(t, bob))
}

func TestClusterDeliversOnceToALocalUser(t *testing.T) {
	mr := miniredis.RunT(t)
	a := replica(t, mr)
	_ = replica(t, mr)
	alice := connect(a, "alice")
	require.Eventually(t, func() bool { return a.IsOnline("alice") }, time.Second, 5*time.Millisecond)

	a.BroadcastRaw("alice", []byte(`{"n":1}`))

	assert.JSONEq(t, `{"n":1}`, receive(t, alice))
	select {
	case extra := <-alice.Send:
		t.Fatalf("delivered twice: %s", extra)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestClusterPresenceIsSharedAndClearedOnDisconnect(t *testing.T) {
	mr := miniredis.RunT(t)
	a, b := replica(t, mr), replica(t, mr)

	assert.False(t, a.IsOnline("bob"))
	bob := connect(b, "bob")
	require.Eventually(t, func() bool { return a.IsOnline("bob") }, time.Second, 5*time.Millisecond,
		"replica A sees a user connected to replica B, so it does not also send them a push")

	b.Unregister(bob)
	require.Eventually(t, func() bool { return !a.IsOnline("bob") }, time.Second, 5*time.Millisecond)
}

func TestExpiredPresenceClaimsAreIgnored(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	c := NewRedisClusterWithClient(rdb)
	ctx := context.Background()

	// A replica that died 2 minutes ago without cleaning up.
	require.NoError(t, rdb.ZAdd(ctx, presencePrefix+"bob", redis.Z{Score: float64(time.Now().Add(-2 * time.Minute).Unix()), Member: "dead"}).Err())
	assert.False(t, c.IsOnline(ctx, "bob"))

	require.NoError(t, c.SetOnline(ctx, "bob"))
	assert.True(t, c.IsOnline(ctx, "bob"))
	ttl := mr.TTL(presencePrefix + "bob")
	assert.True(t, ttl > 0 && ttl <= presenceTTL, "the key expires on its own: %s", ttl)
}

func TestHubFallsBackToLocalDeliveryWithoutRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	a := replica(t, mr)
	alice := connect(a, "alice")
	require.Eventually(t, func() bool { return a.IsOnline("alice") }, time.Second, 5*time.Millisecond)

	mr.Close() // Redis goes away

	a.BroadcastRaw("alice", []byte(`{"still":"here"}`))
	assert.JSONEq(t, `{"still":"here"}`, receive(t, alice), "a replica keeps serving its own connections")
}
