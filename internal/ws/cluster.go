package ws

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cluster lets several chat-api replicas act as one hub.
//
// A hub only knows its own WebSocket connections. With two replicas behind a load balancer
// the sender and the recipient are often connected to different ones: the message was saved
// but never pushed live, and IsOnline said "offline", so the recipient also got an APNs push
// while their app was open. A Cluster fans every delivery out to all replicas and keeps a
// shared view of who is connected.
type Cluster interface {
	// Publish delivers payload to userID's connections on every replica (this one included).
	Publish(ctx context.Context, userID string, payload []byte) error
	// Subscribe calls deliver for every published payload until ctx is done.
	Subscribe(ctx context.Context, deliver func(userID string, payload []byte)) error
	// SetOnline records (or refreshes) that this replica holds connections for users.
	SetOnline(ctx context.Context, userIDs ...string) error
	// SetOffline records that this replica no longer holds connections for userID.
	SetOffline(ctx context.Context, userID string) error
	// IsOnline reports whether any replica holds a connection for userID.
	IsOnline(ctx context.Context, userID string) bool
}

const (
	deliverChannel = "chat:deliver"
	presencePrefix = "chat:online:"
	// presenceTTL is how long a replica's claim lasts without a refresh, so a replica that
	// dies does not leave its users "online" for ever.
	presenceTTL = 60 * time.Second
	// presenceRefresh must be comfortably below presenceTTL.
	presenceRefresh = 20 * time.Second
)

// RedisCluster implements Cluster with Redis pub/sub (deliveries) and one sorted set per
// user (presence: member = replica ID, score = expiry time).
type RedisCluster struct {
	rdb      *redis.Client
	instance string
}

// NewRedisCluster connects to redisURL (redis://host:port/db).
func NewRedisCluster(redisURL string) (*RedisCluster, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("REDIS_URL: %w", err)
	}
	return NewRedisClusterWithClient(redis.NewClient(opts)), nil
}

// NewRedisClusterWithClient wraps an existing client (tests use miniredis).
func NewRedisClusterWithClient(rdb *redis.Client) *RedisCluster {
	return &RedisCluster{rdb: rdb, instance: instanceID()}
}

// Ping checks the connection, for startup.
func (c *RedisCluster) Ping(ctx context.Context) error { return c.rdb.Ping(ctx).Err() }

// Close releases the connection.
func (c *RedisCluster) Close() error { return c.rdb.Close() }

type wireDelivery struct {
	UserID  string          `json:"u"`
	Payload json.RawMessage `json:"p"`
}

// Publish implements Cluster.
func (c *RedisCluster) Publish(ctx context.Context, userID string, payload []byte) error {
	msg, err := json.Marshal(wireDelivery{UserID: userID, Payload: payload})
	if err != nil {
		return err
	}
	return c.rdb.Publish(ctx, deliverChannel, msg).Err()
}

// Subscribe implements Cluster.
func (c *RedisCluster) Subscribe(ctx context.Context, deliver func(userID string, payload []byte)) error {
	sub := c.rdb.Subscribe(ctx, deliverChannel)
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil { // wait for the subscription to be live
		return err
	}
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case m, ok := <-ch:
			if !ok {
				return nil
			}
			var d wireDelivery
			if err := json.Unmarshal([]byte(m.Payload), &d); err != nil || d.UserID == "" {
				continue
			}
			deliver(d.UserID, d.Payload)
		}
	}
}

// SetOnline implements Cluster.
func (c *RedisCluster) SetOnline(ctx context.Context, userIDs ...string) error {
	if len(userIDs) == 0 {
		return nil
	}
	expiry := float64(time.Now().Add(presenceTTL).Unix())
	_, err := c.rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, u := range userIDs {
			p.ZAdd(ctx, presencePrefix+u, redis.Z{Score: expiry, Member: c.instance})
			p.Expire(ctx, presencePrefix+u, presenceTTL)
		}
		return nil
	})
	return err
}

// SetOffline implements Cluster.
func (c *RedisCluster) SetOffline(ctx context.Context, userID string) error {
	return c.rdb.ZRem(ctx, presencePrefix+userID, c.instance).Err()
}

// IsOnline implements Cluster. Claims past their expiry are ignored even before Redis
// drops the key.
func (c *RedisCluster) IsOnline(ctx context.Context, userID string) bool {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	n, err := c.rdb.ZCount(ctx, presencePrefix+userID, now, "+inf").Result()
	return err == nil && n > 0
}

func instanceID() string {
	host, _ := os.Hostname()
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return host + "-" + hex.EncodeToString(b)
}
