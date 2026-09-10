package atsboard_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
)

// leverFixture is a trimmed, representative recording of a real Lever
// postings API response (GET /v0/postings/{company}?mode=json), used
// instead of a live network call.
const leverFixture = `[
  {
    "id": "abc-123",
    "text": "Senior Backend Engineer",
    "categories": {
      "commitment": "Full-time",
      "location": "Remote",
      "team": "Engineering"
    },
    "hostedUrl": "https://jobs.lever.co/acme/abc-123",
    "descriptionPlain": "We are looking for a backend engineer.",
    "description": "<p>We are looking for a backend engineer.</p>"
  }
]`

func TestFetchLeverListings_NormalizesFixtureResponse(t *testing.T) {
	var requestedURL string
	doer := fakeDoer{do: func(req *http.Request) (*http.Response, error) {
		requestedURL = req.URL.String()
		return jsonResponse(http.StatusOK, leverFixture), nil
	}}

	listings, err := atsboard.FetchLeverListings(context.Background(), doer, "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestedURL != "https://api.lever.co/v0/postings/acme?mode=json" {
		t.Errorf("expected request to Lever's postings endpoint for company %q, got %q", "acme", requestedURL)
	}

	if len(listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(listings))
	}
	want := atsboard.Listing{
		Title:       "Senior Backend Engineer",
		Location:    "Remote",
		URL:         "https://jobs.lever.co/acme/abc-123",
		Description: "We are looking for a backend engineer.",
	}
	if listings[0] != want {
		t.Errorf("expected first listing %+v, got %+v", want, listings[0])
	}
}

// Documents issue #42's finding: Lever's public postings API exposes no
// logo field on a posting, so Listing.LogoURL stays empty given a
// real-shaped fixture response with no such field.
func TestFetchLeverListings_NoLogoFieldInResponse_LogoURLStaysEmpty(t *testing.T) {
	doer := fakeDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, leverFixture), nil
	}}

	listings, err := atsboard.FetchLeverListings(context.Background(), doer, "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, l := range listings {
		if l.LogoURL != "" {
			t.Errorf("expected listing %d LogoURL to stay empty, got %q", i, l.LogoURL)
		}
	}
}

func TestFetchLeverListings_BoardNotFound_ReturnsErrBoardNotFound(t *testing.T) {
	doer := fakeDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, `{"ok":false,"error":"Document not found"}`), nil
	}}

	_, err := atsboard.FetchLeverListings(context.Background(), doer, "does-not-exist")
	if !errors.Is(err, atsboard.ErrBoardNotFound) {
		t.Fatalf("expected ErrBoardNotFound, got %v", err)
	}
}
