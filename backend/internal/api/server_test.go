package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// TestNewServer_ConfiguresReadAndIdleTimeouts pins the limits that replace
// http.ListenAndServe's "no limits of any kind" defaults: a client that
// opens a connection and never finishes sending headers, one that stalls
// mid-body, and an idle keep-alive connection all have a bound now.
func TestNewServer_ConfiguresReadAndIdleTimeouts(t *testing.T) {
	dataDir := seedDataDir(t)
	srv := api.NewServer(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}})

	if srv.ReadHeaderTimeout <= 0 {
		t.Errorf("ReadHeaderTimeout must be set, got %v", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout <= 0 {
		t.Errorf("ReadTimeout must be set, got %v", srv.ReadTimeout)
	}
	if srv.IdleTimeout <= 0 {
		t.Errorf("IdleTimeout must be set, got %v", srv.IdleTimeout)
	}
	if srv.ReadHeaderTimeout >= srv.ReadTimeout {
		t.Errorf("ReadHeaderTimeout (%v) should be the tighter of the two read bounds, ReadTimeout is %v", srv.ReadHeaderTimeout, srv.ReadTimeout)
	}
	if srv.Handler == nil {
		t.Error("Handler must be the app router, got nil")
	}
}

// TestNewServer_WriteTimeoutIsDeliberatelyZero defends a deliberate
// absence, which is the whole reason it exists: WriteTimeout is off on
// purpose and this test fails if a future change "fixes" that.
//
// Go's WriteTimeout runs from the end of the header read to the end of the
// response write, so it caps total handler duration. A Generation makes
// four Claude calls (Selection+Rewrite, Cover Letter drafting, then RAL
// estimation's two-call web-search pattern — ADR-0011) and legitimately
// runs for minutes, so any value tight enough to be a useful bound breaks
// the pipeline's core feature. Its failure mode is wrong here too: it
// tears down the connection while the handler keeps running, so the user
// sees a network error and the Claude tokens keep being spent. The bound
// for those routes is a request context deadline instead (see
// withRequestDeadline), which actually stops the outbound work.
func TestNewServer_WriteTimeoutIsDeliberatelyZero(t *testing.T) {
	dataDir := seedDataDir(t)
	srv := api.NewServer(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}})

	if srv.WriteTimeout != 0 {
		t.Fatalf("WriteTimeout must stay 0: it would cap total handler duration and break multi-minute Generations, got %v", srv.WriteTimeout)
	}
}

// TestNewServer_AddrDefaultsAndOverrides covers the one field the caller
// has to be able to set: RouterConfig.Addr, with the documented default
// when it's omitted.
func TestNewServer_AddrDefaultsAndOverrides(t *testing.T) {
	dataDir := seedDataDir(t)

	omitted := api.NewServer(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}})
	if omitted.Addr != api.DefaultAddr {
		t.Errorf("omitted Addr should default to %q, got %q", api.DefaultAddr, omitted.Addr)
	}

	explicit := api.NewServer(api.RouterConfig{DataDir: dataDir, Addr: "127.0.0.1:9999", GenerationClient: &fakeGenerationClient{}})
	if explicit.Addr != "127.0.0.1:9999" {
		t.Errorf("explicit Addr should be used verbatim, got %q", explicit.Addr)
	}
}

// TestNewServer_RequestPathIsUnchanged is the paired check that adding a
// server around the router changed nothing about the requests it serves:
// default (localhost-only, no auth) mode still answers unauthenticated,
// and a configured LAN auth token still rejects a request without it —
// through the handler NewServer built, not one assembled by the test.
func TestNewServer_RequestPathIsUnchanged(t *testing.T) {
	dataDir := seedDataDir(t)

	for _, tc := range []struct {
		name     string
		token    string
		expected int
	}{
		{"default mode, no token configured", "", http.StatusOK},
		{"LAN mode, request carries no token", "s3cret", http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := api.NewServer(api.RouterConfig{
				DataDir:          dataDir,
				GenerationClient: &fakeGenerationClient{},
				LANAuthToken:     tc.token,
			})
			server := httptest.NewServer(srv.Handler)
			defer server.Close()

			resp, err := http.Get(server.URL + "/api/healthz")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.expected {
				t.Fatalf("expected %d, got %d", tc.expected, resp.StatusCode)
			}
		})
	}
}

// TestNewServer_AbandonedRequest_StopsItsClaudeCall is story 12: a browser
// that closes the tab or reloads the page cancels the request, and the
// Claude call behind it stops rather than running on and billing for a
// response nobody will read.
func TestNewServer_AbandonedRequest_StopsItsClaudeCall(t *testing.T) {
	dataDir := seedDataDir(t)
	observed := make(chan error, 1)
	srv := api.NewServer(api.RouterConfig{
		DataDir:          dataDir,
		GenerationClient: blockingGenerationClient(observed),
	})
	server := httptest.NewServer(srv.Handler)
	defer server.Close()

	ctx, abandon := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/generations",
		strings.NewReader(`{"jobDescription":"Looking for a Go backend engineer."}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	done := make(chan struct{})
	go func() {
		defer close(done)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
		}
	}()

	// Give the handler time to reach the (blocking) Claude call, then
	// walk away from it.
	time.Sleep(50 * time.Millisecond)
	abandon()
	<-done

	select {
	case err := <-observed:
		if err != context.Canceled {
			t.Fatalf("expected the outbound call's context to report cancellation, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the outbound call kept running after the request was abandoned")
	}
}
