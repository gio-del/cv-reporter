package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// TestLANAuth_TokenConfigured_RejectsRequestWithoutToken confirms the
// token-check is actually wired into the request path, not just tested in
// isolation: a router built with a LAN auth token configured must reject a
// request that doesn't carry it.
func TestLANAuth_TokenConfigured_RejectsRequestWithoutToken(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithAuth(dataDir, "s3cret"))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// TestLANAuth_NoTokenConfigured_RequestSucceedsUnauthenticated confirms
// default (localhost-only) mode is completely unaffected: with no LAN auth
// token configured, the same request path works with no token at all.
func TestLANAuth_NoTokenConfigured_RequestSucceedsUnauthenticated(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithAuth(dataDir, ""))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
