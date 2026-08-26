package tracking

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SuggestContact asks client to research a Contact for the Job Listing
// identified by id via web search (story 7). It never writes to disk — the
// caller must explicitly PATCH the Application's contact (via
// UpdateApplicationContact) to actually save it, so a suggestion is never
// persisted without confirmation.
func SuggestContact(ctx context.Context, dataDir string, client Client, id string) (Contact, error) {
	listing, err := getJobListing(dataDir, id)
	if err != nil {
		return Contact{}, err
	}
	return client.SuggestContact(ctx, listing.Company, listing.JobDescription)
}

// UpdateApplicationContact validates and saves a Contact to the
// Application identified by id — the explicit confirmation step for both a
// manual entry and an accepted Claude suggestion (story 7).
func UpdateApplicationContact(dataDir, id string, contact Contact) (Application, error) {
	if strings.TrimSpace(contact.Email) == "" {
		return Application{}, fmt.Errorf("%w: email is required", ErrValidation)
	}

	application, err := getApplication(dataDir, id)
	if err != nil {
		return Application{}, err
	}
	application.Contact = &contact

	if err := os.WriteFile(filepath.Join(dataDir, applicationsDir, id+".md"), renderApplication(application), 0o644); err != nil {
		return Application{}, err
	}
	return application, nil
}
