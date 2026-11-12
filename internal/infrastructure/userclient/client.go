// Package userclient calls curiosity-user-api's internal endpoints.
package userclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
		httpClient:  &http.Client{Timeout: 5 * time.Second},
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
