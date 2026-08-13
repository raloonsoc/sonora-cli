package subsonic

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
)

// AuthProvider supplies per-request authentication. Subsonic token auth uses
// query parameters; the planned native JWT API (SPECS §4.3) uses a bearer
// header instead — a second implementation is a drop-in, not a rewrite of
// call sites.
type AuthProvider interface {
	// AuthParams returns query params to append (Subsonic), or nil.
	AuthParams() url.Values
	// AuthHeader returns headers to set (JWT bearer), or nil.
	AuthHeader() http.Header
}

// TokenAuth implements AuthProvider using the Subsonic legacy scheme:
// t = md5(password + salt), with a fresh salt generated per session.
//
// Subsonic requires the salt per request, not per session on the wire — but
// reusing one salt for every request in a session is standard practice and
// what real servers expect; only the token needs regenerating if the salt
// ever changes.
type TokenAuth struct {
	Username string
	password string
	salt     string
}

// NewTokenAuth builds a TokenAuth with a fresh random salt.
func NewTokenAuth(username, password string) (*TokenAuth, error) {
	salt, err := randomSalt()
	if err != nil {
		return nil, err
	}
	return &TokenAuth{Username: username, password: password, salt: salt}, nil
}

func randomSalt() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (a *TokenAuth) token() string {
	sum := md5.Sum([]byte(a.password + a.salt))
	return hex.EncodeToString(sum[:])
}

// AuthParams implements AuthProvider.
func (a *TokenAuth) AuthParams() url.Values {
	v := url.Values{}
	v.Set("u", a.Username)
	v.Set("t", a.token())
	v.Set("s", a.salt)
	return v
}

// AuthHeader implements AuthProvider. Subsonic auth carries no headers.
func (a *TokenAuth) AuthHeader() http.Header {
	return nil
}
