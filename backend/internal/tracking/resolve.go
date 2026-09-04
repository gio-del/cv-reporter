package tracking

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

// Resolve re-attempts RAL Range resolution and/or Application Method
// inference for the Job Listing identified by id, for whichever of the two
// is currently Unresolved — skipping whichever is already resolved
// (stories 9, 12) — and writes back only the field(s) that changed. A field
// that fails again simply stays Unresolved; Resolve itself only errors on
// an unknown id (story 11, 17). Calling it when nothing is Unresolved is a
// safe no-op (story 13).
func Resolve(ctx context.Context, dataDir string, client Client, id string) (JobListing, Application, error) {
	listing, err := getJobListing(dataDir, id)
	if err != nil {
		return JobListing{}, Application{}, err
	}
	application, err := getApplication(dataDir, id)
	if err != nil {
		return JobListing{}, Application{}, err
	}

	if listing.RAL.Source == generation.RALSourceUnresolved {
		listing.RAL = resolveRALBestEffort(ctx, listing.JobDescription, client)
		if err := os.WriteFile(filepath.Join(dataDir, jobsDir, id+".md"), renderJobListing(listing), 0o644); err != nil {
			return JobListing{}, Application{}, err
		}
	}

	if application.Method.Kind == MethodUnresolved {
		application.Method = resolveApplicationMethodBestEffort(ctx, listing.JobDescription, client)
		if err := os.WriteFile(filepath.Join(dataDir, applicationsDir, id+".md"), renderApplication(application), 0o644); err != nil {
			return JobListing{}, Application{}, err
		}
	}

	return listing, application, nil
}
