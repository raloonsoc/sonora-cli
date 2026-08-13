// Package subsonic implements an OpenSubsonic HTTP client: authentication,
// request building, and response decoding for the endpoints in SPECS §4.
package subsonic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	apiVersion     = "1.16.1"
	clientID       = "sonora-cli"
	defaultTimeout = 30 * time.Second
)

// Client is an OpenSubsonic API client bound to one server and one
// AuthProvider.
type Client struct {
	baseURL string
	auth    AuthProvider
	hc      *http.Client
}

// NewClient builds a Client against baseURL, authenticating via auth. hc may
// be nil, in which case a client with a sane default timeout is used.
func NewClient(baseURL string, auth AuthProvider, hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		auth:    auth,
		hc:      hc,
	}
}

// buildURL constructs the request URL for endpoint, merging the fixed
// protocol params, the AuthProvider's params, and any endpoint-specific
// params.
func (c *Client) buildURL(endpoint string, params url.Values) string {
	v := url.Values{}
	v.Set("v", apiVersion)
	v.Set("c", clientID)
	v.Set("f", "json")

	if ap := c.auth.AuthParams(); ap != nil {
		for k, vals := range ap {
			for _, val := range vals {
				v.Add(k, val)
			}
		}
	}
	for k, vals := range params {
		for _, val := range vals {
			v.Add(k, val)
		}
	}

	return fmt.Sprintf("%s/rest/%s?%s", c.baseURL, endpoint, v.Encode())
}

// get performs a GET against endpoint and decodes the subsonic-response
// envelope into resp.
func (c *Client) get(ctx context.Context, endpoint string, params url.Values) (*response, error) {
	reqURL := c.buildURL(endpoint, params)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("subsonic: build request for %s: %w", endpoint, err)
	}
	if h := c.auth.AuthHeader(); h != nil {
		req.Header = h
	}

	res, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("subsonic: %s: %w", endpoint, err)
	}
	defer func() { _ = res.Body.Close() }() // response already fully read below; close error is not actionable

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("subsonic: read %s response: %w", endpoint, err)
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("subsonic: decode %s response: %w", endpoint, err)
	}

	if env.Response.Status == "failed" && env.Response.Error != nil {
		return nil, mapError(env.Response.Error.Code, env.Response.Error.Message)
	}

	return &env.Response, nil
}

// StreamURL returns the signed stream.view URL for track id, for mpv to
// fetch directly — the client never proxies audio bytes (SPECS §5.2).
func (c *Client) StreamURL(id string) string {
	v := url.Values{}
	v.Set("id", id)
	return c.buildURL("stream.view", v)
}

// CoverArtURL returns the signed getCoverArt.view URL for id. size, when
// non-zero, is passed as a hint matched to the render target's dimensions.
func (c *Client) CoverArtURL(id string, size int) string {
	v := url.Values{}
	v.Set("id", id)
	if size > 0 {
		v.Set("size", fmt.Sprintf("%d", size))
	}
	return c.buildURL("getCoverArt.view", v)
}
