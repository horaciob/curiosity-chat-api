package followclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL     string
	internalKey string
	httpClient  *http.Client
}

func NewClient(baseURL, internalKey string) *Client {
	return &Client{
		baseURL:     baseURL,
		internalKey: internalKey,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

type mutualFollowResponse struct {
	Mutual bool `json:"mutual"`
}

func (c *Client) AreFollowing(ctx context.Context, userA, userB string) (bool, error) {
	url := fmt.Sprintf("%s/internal/users/%s/is-mutual-following/%s", c.baseURL, userA, userB)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
		return false, fmt.Errorf("curiosity-api returned status %d", resp.StatusCode)
	}

	var body mutualFollowResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, err
	}
	return body.Mutual, nil
}

type NoopFollowChecker struct{}

func (NoopFollowChecker) AreFollowing(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
