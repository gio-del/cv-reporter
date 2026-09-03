package atsboard

import (
	"context"
	"fmt"
)

// Provider identifies which ATS a board slug belongs to.
type Provider string

const (
	ProviderGreenhouse Provider = "greenhouse"
)

// ErrUnknownProvider is returned by Fetch for a Provider none of
// FetchGreenhouseListings/FetchLeverListings/FetchAshbyListings handle.
var ErrUnknownProvider = fmt.Errorf("atsboard: unknown provider")

// Fetch dispatches to the given Provider's fetch function, so callers (the
// HTTP handler) don't need a switch of their own for every story that adds
// a provider.
func Fetch(ctx context.Context, doer HTTPDoer, provider Provider, boardSlug string) ([]Listing, error) {
	switch provider {
	case ProviderGreenhouse:
		return FetchGreenhouseListings(ctx, doer, boardSlug)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, provider)
	}
}
