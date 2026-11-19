// Package userclient calls curiosity-user-api's internal endpoints.
package userclient

import (
	"sync"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/requestid"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to user-api with the shared internal key.
type Client struct {
	baseURL     string
	internalKey string
	httpClient  *http.Client
}

// NewClient creates a client for user-api at baseURL (e.g. http://user-api:8084).
func NewClient(baseURL, internalKey string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		internalKey: internalKey,
		httpClient:  requestid.NewClient(5 * time.Second),
	}
}

// Names resolves display names for user IDs. Unknown IDs are simply absent.
func (c *Client) Names(ctx context.Context, userIDs []string) (map[string]string, error) {
	if len(userIDs) == 0 {
		return map[string]string{}, nil
	}
	endpoint := c.baseURL + "/internal/users?ids=" + url.QueryEscape(strings.Join(userIDs, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Key", c.internalKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user-api names: status %d", resp.StatusCode)
	}
	var body struct {
		Names map[string]string `json:"names"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Names, nil
}

// ChatPush asks user-api to push a new-message notification to recipientID's devices.
func (c *Client) ChatPush(ctx context.Context, recipientID, senderID, conversationID, text string) error {
	payload, err := json.Marshal(map[string]string{
		"sender_id":       senderID,
		"conversation_id": conversationID,
		"text":            text,
	})
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/internal/users/%s/pushes", c.baseURL, url.PathEscape(recipientID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", c.internalKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("user-api pushes: status %d", resp.StatusCode)
	}
	return nil
}

// AreFriends asks user-api whether two users are friends (GET /internal/friendships).
func (c *Client) AreFriends(ctx context.Context, a, b string) (bool, error) {
	endpoint := c.baseURL + "/internal/friendships?users=" + url.QueryEscape(a+","+b)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("X-Internal-Key", c.internalKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("user-api friendships: status %d", resp.StatusCode)
	}
	var body struct {
		Friends bool `json:"friends"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, err
	}
	return body.Friends, nil
}

// FriendCache remembers friendship answers briefly: a conversation sends many messages
// and each one checks, so user-api is asked at most once per pair per TTL.
type FriendCache struct {
	inner interface {
		AreFriends(ctx context.Context, a, b string) (bool, error)
	}
	ttl time.Duration
	mu  sync.Mutex
	m   map[[2]string]friendEntry
	now func() time.Time
}

type friendEntry struct {
	friends bool
	until   time.Time
}

// NewFriendCache caches inner's answers for ttl.
func NewFriendCache(inner interface {
	AreFriends(ctx context.Context, a, b string) (bool, error)
}, ttl time.Duration) *FriendCache {
	return &FriendCache{inner: inner, ttl: ttl, m: map[[2]string]friendEntry{}, now: time.Now}
}

// AreFriends answers from the cache or asks; errors are not cached.
func (f *FriendCache) AreFriends(ctx context.Context, a, b string) (bool, error) {
	if a > b {
		a, b = b, a
	}
	key := [2]string{a, b}
	f.mu.Lock()
	e, ok := f.m[key]
	f.mu.Unlock()
	if ok && f.now().Before(e.until) {
		return e.friends, nil
	}
	friends, err := f.inner.AreFriends(ctx, a, b)
	if err != nil {
		return false, err
	}
	f.mu.Lock()
	if len(f.m) > 10000 { // bounded: drop everything rather than grow forever
		f.m = map[[2]string]friendEntry{}
	}
	f.m[key] = friendEntry{friends: friends, until: f.now().Add(f.ttl)}
	f.mu.Unlock()
	return friends, nil
}
