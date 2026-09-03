package atsboard_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
)

type fakeDoer struct {
	do func(*http.Request) (*http.Response, error)
}

func (f fakeDoer) Do(req *http.Request) (*http.Response, error) {
	return f.do(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// greenhouseFixture is a trimmed, representative recording of a real
// Greenhouse job board API response (GET
// /v1/boards/{slug}/jobs?content=true), used instead of a live network call.
const greenhouseFixture = `{
  "jobs": [
    {
      "id": 12345,
      "title": "Senior Backend Engineer",
      "location": {"name": "Milan, Italy"},
      "absolute_url": "https://boards.greenhouse.io/acme/jobs/12345",
      "content": "<p>We are looking for a backend engineer.</p>"
    },
    {
      "id": 67890,
      "title": "Product Designer",
      "location": {"name": "Remote"},
      "absolute_url": "https://boards.greenhouse.io/acme/jobs/67890",
      "content": "<p>Design great products.</p>"
    }
  ],
  "meta": {"total": 2}
}`

func TestFetchGreenhouseListings_NormalizesFixtureResponse(t *testing.T) {
	var requestedURL string
	doer := fakeDoer{do: func(req *http.Request) (*http.Response, error) {
		requestedURL = req.URL.String()
		return jsonResponse(http.StatusOK, greenhouseFixture), nil
	}}

	listings, err := atsboard.FetchGreenhouseListings(context.Background(), doer, "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestedURL != "https://boards-api.greenhouse.io/v1/boards/acme/jobs?content=true" {
		t.Errorf("expected request to Greenhouse's board jobs endpoint for slug %q, got %q", "acme", requestedURL)
	}

	if len(listings) != 2 {
		t.Fatalf("expected 2 listings, got %d", len(listings))
	}
	want := atsboard.Listing{
		Title:       "Senior Backend Engineer",
		Location:    "Milan, Italy",
		URL:         "https://boards.greenhouse.io/acme/jobs/12345",
		Description: "<p>We are looking for a backend engineer.</p>",
	}
	if listings[0] != want {
		t.Errorf("expected first listing %+v, got %+v", want, listings[0])
	}
}

func TestFetchGreenhouseListings_BoardNotFound_ReturnsErrBoardNotFound(t *testing.T) {
	doer := fakeDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, `{}`), nil
	}}

	_, err := atsboard.FetchGreenhouseListings(context.Background(), doer, "does-not-exist")
	if !errors.Is(err, atsboard.ErrBoardNotFound) {
		t.Fatalf("expected ErrBoardNotFound, got %v", err)
	}
}
