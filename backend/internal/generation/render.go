package generation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
)

// ErrInvalidRenderRequest marks a Render request that can't be compiled:
// a malformed slug, or approved content citing an Entry Master Data no
// longer has.
var ErrInvalidRenderRequest = errors.New("invalid render request")

var slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// RenderRequest is the approved (and possibly user-edited) content from
// Text Review, ready to compile — see CONTEXT.md's Render entry. Slug
// names the output/<slug>/ directory, mirroring the tailor-cv skill's
// convention (e.g. the company applied to, or "default").
type RenderRequest struct {
	Slug        string
	Selection   SelectionResult
	CoverLetter *CoverLetterResult
}

// RenderResult names the produced PDF(s), relative to outputDir, plus the
// CV's page count so Visual Review can flag overflow (story 10).
type RenderResult struct {
	Slug            string `json:"slug"`
	CVPath          string `json:"cvPath"`
	CoverLetterPath string `json:"coverLetterPath,omitempty"`
	CVPageCount     int    `json:"cvPageCount"`
}

type cvExperience struct {
	Employer string   `json:"employer"`
	Role     string   `json:"role"`
	Client   string   `json:"client,omitempty"`
	Location string   `json:"location,omitempty"`
	Start    string   `json:"start"`
	End      *string  `json:"end"`
	Bullets  []string `json:"bullets"`
}

type cvProject struct {
	Name    string   `json:"name"`
	Repo    string   `json:"repo,omitempty"`
	Start   string   `json:"start"`
	End     *string  `json:"end"`
	Bullets []string `json:"bullets"`
}

type cvData struct {
	Name         string                   `json:"name"`
	Location     string                   `json:"location"`
	Email        string                   `json:"email"`
	Phone        string                   `json:"phone"`
	LinkedIn     string                   `json:"linkedin"`
	GitHub       string                   `json:"github"`
	Education    []masterdata.Education   `json:"education"`
	Experience   []cvExperience           `json:"experience"`
	Projects     []cvProject              `json:"projects"`
	TechStack    []string                 `json:"tech_stack"`
	Publications []masterdata.Publication `json:"publications"`
	Awards       []masterdata.Award       `json:"awards"`
	Activities   []masterdata.Activity    `json:"activities"`
	Languages    []masterdata.Language    `json:"languages"`
}

type coverLetterData struct {
	Name     string `json:"name"`
	Location string `json:"location"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	LinkedIn string `json:"linkedin"`
	GitHub   string `json:"github"`
	Body     string `json:"body"`
}

// Render compiles req into a Tailored CV PDF (and, if req.CoverLetter is
// set, a Cover Letter PDF) under projectRoot/output/<slug>/, invoking the
// typst CLI exactly as the tailor-cv skill does (see CLAUDE.md), so it
// needs projectRoot to contain both template/ and output/ — the same
// "project root" the skill's own `--root .` invocation assumes.
func Render(projectRoot, dataDir string, req RenderRequest) (RenderResult, error) {
	if !slugRe.MatchString(req.Slug) {
		return RenderResult{}, fmt.Errorf("%w: slug must be kebab-case (e.g. \"acme-corp\"), got %q", ErrInvalidRenderRequest, req.Slug)
	}

	profile, err := masterdata.GetProfile(dataDir)
	if err != nil {
		return RenderResult{}, fmt.Errorf("loading profile: %w", err)
	}
	entries, err := masterdata.ListEntries(dataDir)
	if err != nil {
		return RenderResult{}, fmt.Errorf("loading master data: %w", err)
	}
	entriesByID := make(map[string]masterdata.Entry, len(entries))
	for _, e := range entries {
		entriesByID[e.ID] = e
	}

	cv, err := assembleCVData(profile, entriesByID, req.Selection)
	if err != nil {
		return RenderResult{}, err
	}

	outputDir := filepath.Join(projectRoot, "output", req.Slug)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return RenderResult{}, fmt.Errorf("creating output directory: %w", err)
	}

	cvRelPath, err := renderTypst(projectRoot, "template/cv.typ", req.Slug, "data.json", "cv.pdf", cv)
	if err != nil {
		return RenderResult{}, err
	}
	pageCount, err := countPDFPages(filepath.Join(projectRoot, cvRelPath))
	if err != nil {
		return RenderResult{}, fmt.Errorf("counting rendered pages: %w", err)
	}

	result := RenderResult{Slug: req.Slug, CVPath: cvRelPath, CVPageCount: pageCount}

	if req.CoverLetter != nil {
		cl := coverLetterData{
			Name:     profile.Name,
			Location: profile.Location,
			Email:    profile.Email,
			Phone:    profile.Phone,
			LinkedIn: profile.LinkedIn,
			GitHub:   profile.GitHub,
			Body:     req.CoverLetter.Body,
		}
		clRelPath, err := renderTypst(projectRoot, "template/cover-letter.typ", req.Slug, "cover-letter-data.json", "cover-letter.pdf", cl)
		if err != nil {
			return RenderResult{}, err
		}
		result.CoverLetterPath = clRelPath
	}

	return result, nil
}

// renderTypst writes data as JSON to output/<slug>/<dataFile>, then invokes
// `typst compile --root projectRoot template/<template> output/<slug>/<pdfFile>
// --input data=output/<slug>/<dataFile>`, exactly the tailor-cv skill's
// invocation (see CLAUDE.md), and returns the PDF's path relative to
// projectRoot.
func renderTypst(projectRoot, templateRelPath, slug, dataFile, pdfFile string, data any) (string, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshaling render data: %w", err)
	}

	dataRelPath := filepath.Join("output", slug, dataFile)
	if err := os.WriteFile(filepath.Join(projectRoot, dataRelPath), encoded, 0o644); err != nil {
		return "", fmt.Errorf("writing render data: %w", err)
	}

	pdfRelPath := filepath.Join("output", slug, pdfFile)
	cmd := exec.Command("typst", "compile",
		"--root", projectRoot,
		filepath.Join(projectRoot, templateRelPath),
		filepath.Join(projectRoot, pdfRelPath),
		"--input", "data="+dataRelPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("typst compile failed: %w\n%s", err, out)
	}
	return pdfRelPath, nil
}

func assembleCVData(profile masterdata.Profile, entriesByID map[string]masterdata.Entry, selection SelectionResult) (cvData, error) {
	cv := cvData{
		Name:         profile.Name,
		Location:     profile.Location,
		Email:        profile.Email,
		Phone:        profile.Phone,
		LinkedIn:     profile.LinkedIn,
		GitHub:       profile.GitHub,
		Education:    profile.Education,
		Experience:   []cvExperience{},
		Projects:     []cvProject{},
		TechStack:    []string{},
		Publications: profile.Publications,
		Awards:       profile.Awards,
		Activities:   profile.Activities,
		Languages:    profile.Languages,
	}

	seenTag := map[string]bool{}
	for _, se := range selection.Entries {
		if len(se.Bullets) == 0 {
			continue
		}
		entry, ok := entriesByID[se.EntryID]
		if !ok {
			return cvData{}, fmt.Errorf("%w: approved content cites unknown entry id %q", ErrInvalidRenderRequest, se.EntryID)
		}

		bullets := make([]string, len(se.Bullets))
		for i, b := range se.Bullets {
			bullets[i] = b.Rewritten
		}

		switch entry.Type {
		case masterdata.TypeExperience:
			cv.Experience = append(cv.Experience, cvExperience{
				Employer: entry.Employer,
				Role:     entry.Role,
				Client:   entry.Client,
				Location: entry.Location,
				Start:    entry.Start,
				End:      entry.End,
				Bullets:  bullets,
			})
		case masterdata.TypeProject:
			cv.Projects = append(cv.Projects, cvProject{
				Name:    entry.Name,
				Repo:    entry.Repo,
				Start:   entry.Start,
				End:     entry.End,
				Bullets: bullets,
			})
		}

		for _, tag := range entry.Tags {
			if !seenTag[tag] {
				seenTag[tag] = true
				cv.TechStack = append(cv.TechStack, tag)
			}
		}
	}

	return cv, nil
}

// pagesRe matches a PDF's page tree dictionary, e.g. "/Type/Pages/Count 3"
// or "/Count 3/Type/Pages" — typst writes this uncompressed.
var pagesRe = regexp.MustCompile(`/Type\s*/Pages[^<>]{0,200}?/Count\s+(\d+)|/Count\s+(\d+)[^<>]{0,200}?/Type\s*/Pages`)

// countPDFPages reads a rendered PDF's page count directly from its object
// structure — good enough for asserting "the CV is one page" (per the
// PRD's Testing Decisions) without a PDF parsing dependency.
func countPDFPages(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	m := pagesRe.FindSubmatch(content)
	if m == nil {
		return 0, fmt.Errorf("could not find a page count in %s", path)
	}
	for _, group := range m[1:] {
		if len(group) > 0 {
			return strconv.Atoi(string(group))
		}
	}
	return 0, fmt.Errorf("could not find a page count in %s", path)
}
