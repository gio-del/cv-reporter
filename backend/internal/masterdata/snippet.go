package masterdata

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// snippetDir is the directory under dataDir holding Cover Letter Snippet
// files, per ADR-0003's per-Entry-file pattern extended to Snippets (see
// CONTEXT.md's Cover Letter Snippet entry).
const snippetDir = "cover-letter-snippets"

// Snippet is a reusable Cover Letter paragraph (opening, why-this-company,
// closing, ...), stored one-per-file like an Entry.
type Snippet struct {
	ID   string   `json:"id"`
	Kind string   `json:"kind"`
	Tags []string `json:"tags"`
	Body string   `json:"body"`
}

type rawSnippetFrontmatter struct {
	Kind string   `yaml:"kind"`
	Tags []string `yaml:"tags,omitempty"`
}

// ListSnippets reads every Snippet under dataDir/cover-letter-snippets.
func ListSnippets(dataDir string) ([]Snippet, error) {
	fullDir := filepath.Join(dataDir, snippetDir)
	files, err := os.ReadDir(fullDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var snippets []Snippet
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(f.Name(), ".md")
		content, err := os.ReadFile(filepath.Join(fullDir, f.Name()))
		if err != nil {
			return nil, err
		}
		snippet, err := parseSnippet(slug, content)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", f.Name(), err)
		}
		snippets = append(snippets, snippet)
	}
	sort.Slice(snippets, func(i, j int) bool { return snippets[i].ID < snippets[j].ID })
	return snippets, nil
}

// GetSnippet reads a single Snippet by its id (the file's slug).
func GetSnippet(dataDir, id string) (Snippet, error) {
	content, err := os.ReadFile(filepath.Join(dataDir, snippetDir, id+".md"))
	if err != nil {
		return Snippet{}, err
	}
	return parseSnippet(id, content)
}

func parseSnippet(slug string, content []byte) (Snippet, error) {
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return Snippet{}, err
	}

	var raw rawSnippetFrontmatter
	if err := yaml.Unmarshal(fm, &raw); err != nil {
		return Snippet{}, err
	}

	return Snippet{
		ID:   slug,
		Kind: raw.Kind,
		Tags: raw.Tags,
		Body: strings.TrimSpace(string(body)),
	}, nil
}

// CreateSnippet validates snippet and, if valid, writes it to a new file
// under dataDir/cover-letter-snippets, generating a slug (and thus id) from
// its kind.
func CreateSnippet(dataDir string, snippet Snippet) (Snippet, error) {
	if err := ValidateSnippet(snippet); err != nil {
		return Snippet{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	fullDir := filepath.Join(dataDir, snippetDir)
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return Snippet{}, err
	}

	slug := uniqueSlug(fullDir, slugify(snippet.Kind))
	snippet.ID = slug

	if err := os.WriteFile(filepath.Join(fullDir, slug+".md"), renderSnippet(snippet), 0o644); err != nil {
		return Snippet{}, err
	}
	return snippet, nil
}

// UpdateSnippet validates snippet and, if valid, writes it back to the same
// file GetSnippet would read for id. On validation failure the file is left
// untouched.
func UpdateSnippet(dataDir, id string, snippet Snippet) (Snippet, error) {
	path := filepath.Join(dataDir, snippetDir, id+".md")
	if _, err := os.Stat(path); err != nil {
		return Snippet{}, err
	}

	snippet.ID = id

	if err := ValidateSnippet(snippet); err != nil {
		return Snippet{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	if err := os.WriteFile(path, renderSnippet(snippet), 0o644); err != nil {
		return Snippet{}, err
	}
	return snippet, nil
}

// DeleteSnippet removes the file GetSnippet would read for id.
func DeleteSnippet(dataDir, id string) error {
	return os.Remove(filepath.Join(dataDir, snippetDir, id+".md"))
}

// ValidateSnippet checks that a Snippet's required fields are present before
// it is written to disk.
func ValidateSnippet(s Snippet) error {
	if s.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	if strings.TrimSpace(s.Body) == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}

func renderSnippet(s Snippet) []byte {
	raw := rawSnippetFrontmatter{Kind: s.Kind, Tags: s.Tags}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	fmBytes, _ := yaml.Marshal(raw)
	buf.Write(fmBytes)
	buf.WriteString("---\n\n")
	buf.WriteString(s.Body)
	buf.WriteString("\n")
	return buf.Bytes()
}
