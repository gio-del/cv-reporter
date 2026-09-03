package atsboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type greenhouseJob struct {
	Title    string `json:"title"`
	Location struct {
		Name string `json:"name"`
	} `json:"location"`
	AbsoluteURL string `json:"absolute_url"`
	Content     string `json:"content"`
}

type greenhouseResponse struct {
	Jobs []greenhouseJob `json:"jobs"`
}

// FetchGreenhouseListings fetches boardSlug's open roles from Greenhouse's
// public job board API (documented, read-only — see ADR-0007) and
// normalizes them into the common Listing shape (story 1).
func FetchGreenhouseListings(ctx context.Context, doer HTTPDoer, boardSlug string) ([]Listing, error) {
	url := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", boardSlug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building Greenhouse request: %w", err)
	}

	resp, err := doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Greenhouse board %q: %w", boardSlug, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("Greenhouse board %q: %w", boardSlug, ErrBoardNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching Greenhouse board %q: got status %d", boardSlug, resp.StatusCode)
	}

	var parsed greenhouseResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("parsing Greenhouse response for board %q: %w", boardSlug, err)
	}

	listings := make([]Listing, 0, len(parsed.Jobs))
	for _, j := range parsed.Jobs {
		listings = append(listings, Listing{
			Title:       j.Title,
			Location:    j.Location.Name,
			URL:         j.AbsoluteURL,
			Description: j.Content,
		})
	}
	return listings, nil
}
