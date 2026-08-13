// Package nativeapi implements sonora's native JWT auth API (SPECS §4.3),
// a planned non-plaintext-password alternative to Subsonic token auth.
package nativeapi

import (
	"net/http"
	"net/url"
	"sync"
)

// BearerAuth implements subsonic.AuthProvider using an
// "Authorization: Bearer <token>" header instead of Subsonic's query-param
// token scheme. It holds the access token in memory only — nothing here is
// ever persisted; callers persist the refresh token separately via Client.
//
// The access token is set by Client after login/refresh and read on every
// request, so a background refresh (SPECS §4.3: "before each session or on
// a 401") takes effect on the very next call without the caller having to
// rebuild anything.
type BearerAuth struct {
	mu          sync.RWMutex
	accessToken string
}

// SetAccessToken updates the in-memory access token used on subsequent
// requests. Safe to call concurrently with AuthHeader.
func (a *BearerAuth) SetAccessToken(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.accessToken = token
}

// AuthParams implements subsonic.AuthProvider. Native auth carries no query
// params.
func (a *BearerAuth) AuthParams() url.Values {
	return nil
}

// AuthHeader implements subsonic.AuthProvider.
func (a *BearerAuth) AuthHeader() http.Header {
	a.mu.RLock()
	defer a.mu.RUnlock()

	h := http.Header{}
	h.Set("Authorization", "Bearer "+a.accessToken)
	return h
}
