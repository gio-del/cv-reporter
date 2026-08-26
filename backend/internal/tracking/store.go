package tracking

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
	"gopkg.in/yaml.v3"
)

const (
	jobsDir         = "jobs"
	applicationsDir = "applications"
)

// ErrValidation marks a SaveRequest that can't be saved.
var ErrValidation = errors.New("validation failed")

// SaveRequest is the input to Save: a Company name plus a Job Description,
// pasted verbatim or fetched from a URL — mirroring
// generation.GenerateRequest's JobDescription/JobDescriptionURL shape
// (story 1).
type SaveRequest struct {
	Company           string
	URL               string
	JobDescription    string
	JobDescriptionURL string
}

type rawJobListingFrontmatter struct {
	Company string              `yaml:"company"`
	URL     string              `yaml:"url,omitempty"`
	Source  string              `yaml:"source"`
	SavedAt string              `yaml:"savedAt"`
	RAL     generation.RALRange `yaml:"ral"`
}

type rawApplication struct {
	JobListingID string `yaml:"jobListingId"`
	Status       Status `yaml:"status"`
}

// Save resolves req's Job Description and RAL Range, then writes a new Job
// Listing and its linked Application (Status Saved) to dataDir — "saving a
// Job Listing immediately creates its Application" (story 2).
func Save(ctx context.Context, dataDir string, client generation.Client, req SaveRequest) (JobListing, Application, error) {
	if strings.TrimSpace(req.Company) == "" {
		return JobListing{}, Application{}, fmt.Errorf("%w: company is required", ErrValidation)
	}

	jobDescription, err := generation.ResolveJobDescription(ctx, req.JobDescription, req.JobDescriptionURL)
	if err != nil {
		return JobListing{}, Application{}, err
	}
	if jobDescription == "" {
		return JobListing{}, Application{}, fmt.Errorf("%w: jobDescription or jobDescriptionUrl is required", ErrValidation)
	}

	ral, err := generation.ResolveRAL(ctx, jobDescription, client)
	if err != nil {
		return JobListing{}, Application{}, fmt.Errorf("resolving RAL range: %w", err)
	}

	jobsFullDir := filepath.Join(dataDir, jobsDir)
	if err := os.MkdirAll(jobsFullDir, 0o755); err != nil {
		return JobListing{}, Application{}, err
	}
	slug := uniqueSlug(jobsFullDir, slugify(req.Company))

	listing := JobListing{
		ID:             slug,
		Company:        req.Company,
		URL:            req.URL,
		Source:         SourceManual,
		SavedAt:        time.Now().UTC().Format(time.RFC3339),
		JobDescription: jobDescription,
		RAL:            ral,
	}
	if err := os.WriteFile(filepath.Join(jobsFullDir, slug+".md"), renderJobListing(listing), 0o644); err != nil {
		return JobListing{}, Application{}, err
	}

	applicationsFullDir := filepath.Join(dataDir, applicationsDir)
	if err := os.MkdirAll(applicationsFullDir, 0o755); err != nil {
		return JobListing{}, Application{}, err
	}
	application := Application{ID: slug, JobListingID: slug, Status: StatusSaved}
	if err := os.WriteFile(filepath.Join(applicationsFullDir, slug+".md"), renderApplication(application), 0o644); err != nil {
		return JobListing{}, Application{}, err
	}

	return listing, application, nil
}

func renderJobListing(l JobListing) []byte {
	raw := rawJobListingFrontmatter{
		Company: l.Company,
		URL:     l.URL,
		Source:  l.Source,
		SavedAt: l.SavedAt,
		RAL:     l.RAL,
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	fmBytes, _ := yaml.Marshal(raw)
	buf.Write(fmBytes)
	buf.WriteString("---\n\n")
	buf.WriteString(l.JobDescription)
	buf.WriteString("\n")
	return buf.Bytes()
}

func renderApplication(a Application) []byte {
	raw := rawApplication{JobListingID: a.JobListingID, Status: a.Status}
	out, _ := yaml.Marshal(raw)
	return out
}

func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// uniqueSlug appends -2, -3, ... to base until it no longer collides with an
// existing Job Listing file in fullDir.
func uniqueSlug(fullDir, base string) string {
	if base == "" {
		base = "job"
	}
	slug := base
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(fullDir, slug+".md")); os.IsNotExist(err) {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}
