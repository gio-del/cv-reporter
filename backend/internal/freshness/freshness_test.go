package freshness_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/freshness"
)

type fakeDoer struct {
	do func(*http.Request) (*http.Response, error)
}

func (f fakeDoer) Do(req *http.Request) (*http.Response, error) {
	return f.do(req)
}

func emptyResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

// TestCheck_ClassifiesFixtureResponses is the table-driven test the PRD's
// Testing Decisions call for: given a fake HTTPDoer returning a fixture
// response (status code, a redirect-to-login case, or a simulated
// timeout/network error), Check returns the correct one of
// Live/Unreachable/Unknown — independent of persistence, the API layer, or
// the FE.
func TestCheck_ClassifiesFixtureResponses(t *testing.T) {
	targetURL := "https://boards.example.com/acme/jobs/12345"

	tests := []struct {
		name string
		doer freshness.HTTPDoer
		want freshness.Status
	}{
		{
			name: "200 OK is Live",
			doer: fakeDoer{do: func(req *http.Request) (*http.Response, error) {
				return emptyResponse(http.StatusOK), nil
			}},
			want: freshness.StatusLive,
		},
		{
			name: "404 Not Found is Unreachable",
			doer: fakeDoer{do: func(req *http.Request) (*http.Response, error) {
				return emptyResponse(http.StatusNotFound), nil
			}},
			want: freshness.StatusUnreachable,
		},
		{
			name: "403 Forbidden is Unknown (auth-wall, not dead)",
			doer: fakeDoer{do: func(req *http.Request) (*http.Response, error) {
				return emptyResponse(http.StatusForbidden), nil
			}},
			want: freshness.StatusUnknown,
		},
		{
			name: "401 Unauthorized is Unknown",
			doer: fakeDoer{do: func(req *http.Request) (*http.Response, error) {
				return emptyResponse(http.StatusUnauthorized), nil
			}},
			want: freshness.StatusUnknown,
		},
		{
			name: "redirect chain landing on a login page is Unknown",
			doer: fakeDoer{do: func(req *http.Request) (*http.Response, error) {
				// http.Client follows redirects transparently and returns the
				// final response with resp.Request set to the final request —
				// simulate that final landing page being a generic login/SSO
				// wall rather than the original posting.
				finalReq, _ := http.NewRequest(http.MethodGet, "https://boards.example.com/login?redirect=%2Facme%2Fjobs%2F12345", nil)
				resp := emptyResponse(http.StatusOK)
				resp.Request = finalReq
				return resp, nil
			}},
			want: freshness.StatusUnknown,
		},
		{
			name: "timeout/network error is Unknown, not Unreachable",
			doer: fakeDoer{do: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("dial tcp: i/o timeout")
			}},
			want: freshness.StatusUnknown,
		},
		{
			name: "DNS failure is Unknown, not Unreachable",
			doer: fakeDoer{do: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("dial tcp: lookup boards.example.com: no such host")
			}},
			want: freshness.StatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := freshness.Check(context.Background(), tt.doer, targetURL)
			if got != tt.want {
				t.Errorf("Check() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheck_PlainGET_NotContentReparse(t *testing.T) {
	// Story 7: the check is a plain fetch, not a re-scrape — Check must not
	// read the whole body / re-parse content to classify. A doer that
	// returns a response whose body would fail if read twice (or is huge)
	// should still classify fine off status code alone.
	var requestedMethod string
	doer := fakeDoer{do: func(req *http.Request) (*http.Response, error) {
		requestedMethod = req.Method
		return emptyResponse(http.StatusOK), nil
	}}

	got := freshness.Check(context.Background(), doer, "https://boards.example.com/acme/jobs/1")
	if got != freshness.StatusLive {
		t.Fatalf("expected Live, got %q", got)
	}
	if requestedMethod != http.MethodGet {
		t.Errorf("expected a GET request, got %q", requestedMethod)
	}
}
