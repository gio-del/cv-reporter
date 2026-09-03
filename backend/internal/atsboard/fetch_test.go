package atsboard_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/atsboard"
)

func TestFetch_UnknownProvider_ReturnsErrUnknownProvider(t *testing.T) {
	doer := fakeDoer{do: func(req *http.Request) (*http.Response, error) {
		t.Fatal("expected no HTTP call for an unknown provider")
		return nil, nil
	}}

	_, err := atsboard.Fetch(context.Background(), doer, atsboard.Provider("workday"), "acme")
	if !errors.Is(err, atsboard.ErrUnknownProvider) {
		t.Fatalf("expected ErrUnknownProvider, got %v", err)
	}
}

func TestFetch_DispatchesToLever(t *testing.T) {
	doer := fakeDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, leverFixture), nil
	}}

	listings, err := atsboard.Fetch(context.Background(), doer, atsboard.ProviderLever, "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(listings))
	}
}
