package followclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client checks follow relationships by calling curiosity-api.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new follow checker HTTP client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type followersResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// AreFollowing returns true if userA and userB mutually follow each other (both accepted).
func (c *Client) AreFollowing(ctx context.Context, userA, userB string) (bool, error) {
	aFollowsB, err := c.isInFollowers(ctx, userB, userA)
	if err != nil {
		return false, fmt.Errorf("checking if %s follows %s: %w", userA, userB, err)
	}
	if !aFollowsB {
		return false, nil
	}

	bFollowsA, err := c.isInFollowers(ctx, userA, userB)
	if err != nil {
		return false, fmt.Errorf("checking if %s follows %s: %w", userB, userA, err)
	}

	return bFollowsA, nil
}

// isInFollowers checks whether followerID appears in the follower list of userID.
// It uses page[limit]=100 as a practical upper bound for the first page.
func (c *Client) isInFollowers(ctx context.Context, userID, followerID string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/users/%s/followers?page[limit]=100", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("curiosity-api returned status %d", resp.StatusCode)
	}

	var body followersResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, err
	}

	for _, item := range body.Data {
		if item.ID == followerID {
			return true, nil
		}
	}
	return false, nil
}

// NoopFollowChecker always allows chat (for development/testing).
type NoopFollowChecker struct{}

func (NoopFollowChecker) AreFollowing(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
