package atsboard_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
)

// ashbyFixture is a trimmed, representative recording of a real Ashby job
// board API response (GET /posting-api/job-board/{slug}), used instead of
// a live network call.
const ashbyFixture = `{
  "jobs": [
    {
      "id": "7458d4e9-da2e-47bd-98cb-adfda43d42b2",
      "title": "Engineering Manager",
      "location": "Remote - European Union",
      "jobUrl": "https://jobs.ashbyhq.com/acme/7458d4e9-da2e-47bd-98cb-adfda43d42b2",
      "descriptionPlain": "We are looking for an engineering manager.",
      "descriptionHtml": "<p>We are looking for an engineering manager.</p>"
    }
  ]
}`

func TestFetchAshbyListings_NormalizesFixtureResponse(t *testing.T) {
	var requestedURL string
	doer := fakeDoer{do: func(req *http.Request) (*http.Response, error) {
		requestedURL = req.URL.String()
		return jsonResponse(http.StatusOK, ashbyFixture), nil
	}}

	listings, err := atsboard.FetchAshbyListings(context.Background(), doer, "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestedURL != "https://api.ashbyhq.com/posting-api/job-board/acme" {
		t.Errorf("expected request to Ashby's job board endpoint for slug %q, got %q", "acme", requestedURL)
	}

	if len(listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(listings))
	}
	want := atsboard.Listing{
		Title:       "Engineering Manager",
		Location:    "Remote - European Union",
		URL:         "https://jobs.ashbyhq.com/acme/7458d4e9-da2e-47bd-98cb-adfda43d42b2",
		Description: "We are looking for an engineering manager.",
	}
	if listings[0] != want {
		t.Errorf("expected first listing %+v, got %+v", want, listings[0])
	}
}

func TestFetchAshbyListings_BoardNotFound_ReturnsErrBoardNotFound(t *testing.T) {
	doer := fakeDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, `{}`), nil
	}}

	_, err := atsboard.FetchAshbyListings(context.Background(), doer, "does-not-exist")
	if !errors.Is(err, atsboard.ErrBoardNotFound) {
		t.Fatalf("expected ErrBoardNotFound, got %v", err)
	}
}
