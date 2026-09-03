package atsboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type leverPosting struct {
	Text       string `json:"text"`
	Categories struct {
		Location string `json:"location"`
	} `json:"categories"`
	HostedURL        string `json:"hostedUrl"`
	DescriptionPlain string `json:"descriptionPlain"`
}

// FetchLeverListings fetches company's open roles from Lever's public
// postings API (documented, read-only — see ADR-0007) and normalizes them
// into the common Listing shape (story 2).
func FetchLeverListings(ctx context.Context, doer HTTPDoer, company string) ([]Listing, error) {
	url := fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", company)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building Lever request: %w", err)
	}

	resp, err := doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Lever company %q: %w", company, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("Lever company %q: %w", company, ErrBoardNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching Lever company %q: got status %d", company, resp.StatusCode)
	}

	var postings []leverPosting
	if err := json.NewDecoder(resp.Body).Decode(&postings); err != nil {
		return nil, fmt.Errorf("parsing Lever response for company %q: %w", company, err)
	}

	listings := make([]Listing, 0, len(postings))
	for _, p := range postings {
		listings = append(listings, Listing{
			Title:       p.Text,
			Location:    p.Categories.Location,
			URL:         p.HostedURL,
			Description: p.DescriptionPlain,
		})
	}
	return listings, nil
}
