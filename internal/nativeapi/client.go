package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultTimeout = 30 * time.Second

// Client speaks sonora's native JWT auth API. It owns a BearerAuth,
// updating its in-memory access token on every successful login or
// refresh — callers wire the same BearerAuth into a subsonic.Client (or an
// equivalent native-API resource client) to pick up each refresh
// automatically.
type Client struct {
	baseURL string
	hc      *http.Client
	auth    *BearerAuth

	mu           sync.Mutex
	refreshToken string
}

// NewClient builds a Client against baseURL, updating auth's access token
// on every login/refresh. hc may be nil, in which case a client with a
// sane default timeout is used.
func NewClient(baseURL string, auth *BearerAuth, hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		hc:      hc,
		auth:    auth,
	}
}

// tokenPair mirrors the {access_token, refresh_token} shape returned by
// both login and refresh (SPECS §4.3).
type tokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Login authenticates with username/password and stores the resulting
// access token in the wired BearerAuth and the refresh token in Client, for
// the caller to persist via RefreshToken.
func (c *Client) Login(ctx context.Context, username, password string) error {
	body := map[string]string{"username": username, "password": password}
	pair, err := c.post(ctx, "/api/auth/login", body)
	if err != nil {
		return fmt.Errorf("nativeapi: login: %w", err)
	}
	c.storeTokens(pair)
	return nil
}

// Refresh exchanges the current refresh token for a new access/refresh
// pair, rotating the stored refresh token (SPECS §4.3: "Rotated on every
// use"). Call this transparently before a session starts or after a 401.
func (c *Client) Refresh(ctx context.Context) error {
	c.mu.Lock()
	rt := c.refreshToken
	c.mu.Unlock()

	if rt == "" {
		return fmt.Errorf("nativeapi: refresh: no refresh token available")
	}

	pair, err := c.post(ctx, "/api/auth/refresh", map[string]string{"refresh_token": rt})
	if err != nil {
		return fmt.Errorf("nativeapi: refresh: %w", err)
	}
	c.storeTokens(pair)
	return nil
}

// Logout revokes the current refresh token server-side.
func (c *Client) Logout(ctx context.Context) error {
	c.mu.Lock()
	rt := c.refreshToken
	c.mu.Unlock()

	if rt == "" {
		return nil // nothing to revoke
	}

	req, err := c.newRequest(ctx, "/api/auth/logout", map[string]string{"refresh_token": rt})
	if err != nil {
		return fmt.Errorf("nativeapi: logout: %w", err)
	}
	res, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("nativeapi: logout: %w", err)
	}
	defer func() { _ = res.Body.Close() }() // no response body expected on 204

	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("nativeapi: logout: unexpected status %s", res.Status)
	}
	return nil
}

// RefreshToken returns the current refresh token, for the caller to
// persist (keychain or config file) — the access token is never persisted
// (SPECS §4.3).
func (c *Client) RefreshToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.refreshToken
}

// SetRefreshToken restores a previously persisted refresh token, e.g. on
// startup before calling Refresh to obtain a fresh access token.
func (c *Client) SetRefreshToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refreshToken = token
}

func (c *Client) storeTokens(pair tokenPair) {
	c.mu.Lock()
	c.refreshToken = pair.RefreshToken
	c.mu.Unlock()
	c.auth.SetAccessToken(pair.AccessToken)
}

func (c *Client) newRequest(ctx context.Context, path string, body any) (*http.Request, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *Client) post(ctx context.Context, path string, body any) (tokenPair, error) {
	req, err := c.newRequest(ctx, path, body)
	if err != nil {
		return tokenPair{}, err
	}

	res, err := c.hc.Do(req)
	if err != nil {
		return tokenPair{}, err
	}
	defer func() { _ = res.Body.Close() }() // response fully read below; close error is not actionable

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return tokenPair{}, fmt.Errorf("read response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return tokenPair{}, fmt.Errorf("unexpected status %s", res.Status)
	}

	var pair tokenPair
	if err := json.Unmarshal(respBody, &pair); err != nil {
		return tokenPair{}, fmt.Errorf("decode response: %w", err)
	}
	return pair, nil
}
