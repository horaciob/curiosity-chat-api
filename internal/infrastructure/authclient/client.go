package authclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client calls curiosity-auth-api for token validation.
type Client struct {
	baseURL     string
	internalKey string
	httpClient  *http.Client
}

func NewClient(baseURL, internalKey string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		internalKey: internalKey,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

// Validate calls POST /internal/token/validate on auth-api.
// It satisfies the middleware.TokenValidator interface.
func (c *Client) Validate(bearerToken string) (string, error) {
	return c.ValidateCtx(context.Background(), bearerToken)
}

func (c *Client) ValidateCtx(ctx context.Context, bearerToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/internal/token/validate", c.baseURL), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Internal-Key", c.internalKey)
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", errors.New("invalid or expired token")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth-api returned status %d", resp.StatusCode)
	}

	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.UserID == "" {
		return "", errors.New("auth-api returned empty user_id")
	}
	return body.UserID, nil
}
