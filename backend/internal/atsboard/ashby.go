package atsboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type ashbyJob struct {
	Title            string `json:"title"`
	Location         string `json:"location"`
	JobURL           string `json:"jobUrl"`
	DescriptionPlain string `json:"descriptionPlain"`
}

type ashbyResponse struct {
	Jobs []ashbyJob `json:"jobs"`
}

// FetchAshbyListings fetches boardSlug's open roles from Ashby's public job
// board API (documented, read-only — see ADR-0007) and normalizes them
// into the common Listing shape (story 3).
func FetchAshbyListings(ctx context.Context, doer HTTPDoer, boardSlug string) ([]Listing, error) {
	url := fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", boardSlug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building Ashby request: %w", err)
	}

	resp, err := doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Ashby board %q: %w", boardSlug, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("Ashby board %q: %w", boardSlug, ErrBoardNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching Ashby board %q: got status %d", boardSlug, resp.StatusCode)
	}

	var parsed ashbyResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("parsing Ashby response for board %q: %w", boardSlug, err)
	}

	listings := make([]Listing, 0, len(parsed.Jobs))
	for _, j := range parsed.Jobs {
		listings = append(listings, Listing{
			Title:       j.Title,
			Location:    j.Location,
			URL:         j.JobURL,
			Description: j.DescriptionPlain,
		})
	}
	return listings, nil
}
