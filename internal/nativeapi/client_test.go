package nativeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Login(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/login" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["username"] != "alice" || body["password"] != "hunter2" {
			t.Errorf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at-1","refresh_token":"rt-1"}`))
	}))
	defer srv.Close()

	auth := &BearerAuth{}
	c := NewClient(srv.URL, auth, srv.Client())

	if err := c.Login(context.Background(), "alice", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	if got := auth.AuthHeader().Get("Authorization"); got != "Bearer at-1" {
		t.Errorf("access token header = %q, want Bearer at-1", got)
	}
	if got := c.RefreshToken(); got != "rt-1" {
		t.Errorf("RefreshToken() = %q, want rt-1", got)
	}
}

func TestClient_Refresh_rotatesToken(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["refresh_token"] != "rt-1" {
			t.Errorf("refresh called with %q, want rt-1", body["refresh_token"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at-2","refresh_token":"rt-2"}`))
	}))
	defer srv.Close()

	auth := &BearerAuth{}
	c := NewClient(srv.URL, auth, srv.Client())
	c.SetRefreshToken("rt-1")

	if err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if got := c.RefreshToken(); got != "rt-2" {
		t.Errorf("RefreshToken() after refresh = %q, want rt-2 (rotated)", got)
	}
	if got := auth.AuthHeader().Get("Authorization"); got != "Bearer at-2" {
		t.Errorf("access token header = %q, want Bearer at-2", got)
	}
	if callCount != 1 {
		t.Errorf("server called %d times, want 1", callCount)
	}
}

func TestClient_Refresh_noTokenIsAnError(t *testing.T) {
	auth := &BearerAuth{}
	c := NewClient("https://example.com", auth, nil)

	if err := c.Refresh(context.Background()); err == nil {
		t.Fatal("expected an error when no refresh token has been set")
	}
}

func TestClient_Logout(t *testing.T) {
	var gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotToken = body["refresh_token"]
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	auth := &BearerAuth{}
	c := NewClient(srv.URL, auth, srv.Client())
	c.SetRefreshToken("rt-1")

	if err := c.Logout(context.Background()); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if gotToken != "rt-1" {
		t.Errorf("logout sent refresh_token %q, want rt-1", gotToken)
	}
}

func TestClient_Logout_noTokenIsNoOp(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	auth := &BearerAuth{}
	c := NewClient(srv.URL, auth, srv.Client())

	if err := c.Logout(context.Background()); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if called {
		t.Error("expected no request when there's no refresh token to revoke")
	}
}

func TestBearerAuth_AuthParamsIsNil(t *testing.T) {
	auth := &BearerAuth{}
	if params := auth.AuthParams(); params != nil {
		t.Errorf("AuthParams() = %v, want nil", params)
	}
}
