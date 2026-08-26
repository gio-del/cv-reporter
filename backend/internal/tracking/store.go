package tracking

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
		SavedAt:        time.Now().UTC().Format(time.RFC3339Nano),
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

// List reads every Job Listing under dataDir/jobs, paired with its 1:1
// Application, newest-saved first — the pipeline-at-a-glance view (story 3).
func List(dataDir string) ([]ListingWithApplication, error) {
	jobsFullDir := filepath.Join(dataDir, jobsDir)
	files, err := os.ReadDir(jobsFullDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var result []ListingWithApplication
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(f.Name(), ".md")

		listing, err := getJobListing(dataDir, slug)
		if err != nil {
			return nil, fmt.Errorf("reading job listing %s: %w", f.Name(), err)
		}
		application, err := getApplication(dataDir, slug)
		if err != nil {
			return nil, fmt.Errorf("reading application for job listing %s: %w", slug, err)
		}
		result = append(result, ListingWithApplication{JobListing: listing, Application: application})
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].JobListing.SavedAt > result[j].JobListing.SavedAt
	})
	return result, nil
}

func getJobListing(dataDir, slug string) (JobListing, error) {
	content, err := os.ReadFile(filepath.Join(dataDir, jobsDir, slug+".md"))
	if err != nil {
		return JobListing{}, err
	}
	return parseJobListing(slug, content)
}

func parseJobListing(slug string, content []byte) (JobListing, error) {
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return JobListing{}, err
	}
	var raw rawJobListingFrontmatter
	if err := yaml.Unmarshal(fm, &raw); err != nil {
		return JobListing{}, err
	}
	return JobListing{
		ID:             slug,
		Company:        raw.Company,
		URL:            raw.URL,
		Source:         raw.Source,
		SavedAt:        raw.SavedAt,
		JobDescription: strings.TrimSpace(string(body)),
		RAL:            raw.RAL,
	}, nil
}

func getApplication(dataDir, slug string) (Application, error) {
	content, err := os.ReadFile(filepath.Join(dataDir, applicationsDir, slug+".md"))
	if err != nil {
		return Application{}, err
	}
	var raw rawApplication
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return Application{}, err
	}
	return Application{ID: slug, JobListingID: raw.JobListingID, Status: raw.Status}, nil
}

// splitFrontmatter splits a Job Listing file's content into its YAML
// frontmatter and Markdown body (its Job Description), mirroring
// masterdata's Entry file shape (ADR-0003 extended to Job Listings by
// ADR-0008).
func splitFrontmatter(content []byte) (frontmatter, body []byte, err error) {
	trimmed := bytes.TrimLeft(content, "\n")
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return nil, nil, fmt.Errorf("missing frontmatter delimiter")
	}
	rest := trimmed[len("---"):]
	rest = bytes.TrimPrefix(rest, []byte("\r\n"))
	rest = bytes.TrimPrefix(rest, []byte("\n"))

	idx := bytes.Index(rest, []byte("\n---"))
	if idx == -1 {
		return nil, nil, fmt.Errorf("missing closing frontmatter delimiter")
	}
	frontmatter = rest[:idx]

	after := rest[idx+len("\n---"):]
	after = bytes.TrimPrefix(after, []byte("\r\n"))
	after = bytes.TrimPrefix(after, []byte("\n"))
	body = after
	return frontmatter, body, nil
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
