package tracking

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// MigrationOptions configures MigrateRecords.
type MigrationOptions struct {
	// DryRun computes and reports every change without writing any file.
	DryRun bool
}

// RecordKind names which of the two record types a migration touched.
type RecordKind string

const (
	RecordJobListing  RecordKind = "job listing"
	RecordApplication RecordKind = "application"
)

// MigrationReport is what MigrateRecords did, or in a dry run would do.
type MigrationReport struct {
	DryRun bool
	// Scanned counts every Job Listing and Application file read.
	Scanned int
	// Migrated names each record that was (or would be) rewritten, in the
	// order it was scanned: Job Listings first, then Applications.
	Migrated []RecordMigration
	// Inconsistencies are problems found alongside the migration that it
	// neither fixes nor halts on, such as a Job Listing with no Application.
	Inconsistencies []Inconsistency
}

// Pending reports whether any record is (or, before a write run, was) not
// at CurrentSchemaVersion.
func (r MigrationReport) Pending() bool {
	return len(r.Migrated) > 0
}

// RecordMigration is one record brought to CurrentSchemaVersion.
type RecordMigration struct {
	// Path is relative to the data directory, slash-separated
	// ("jobs/acme.md").
	Path        string
	Kind        RecordKind
	FromVersion int
	ToVersion   int
	// Backfilled are the fields written because their value is provably
	// recoverable from what is already on disk.
	Backfilled []BackfilledField
	// Unknowable are the fields deliberately left empty because nothing on
	// disk records them. They stay empty for good.
	Unknowable []UnknowableField
}

// BackfilledField is a field MigrateRecords filled, and the value it wrote.
type BackfilledField struct {
	Field string
	Value string
}

// UnknowableField is a field MigrateRecords refused to fill, and why.
type UnknowableField struct {
	Field  string
	Reason string
}

// Inconsistency is a problem with a record MigrateRecords reports but does
// not act on.
type Inconsistency struct {
	Path    string
	Problem string
}

// migrationWrite is a planned rewrite of one file.
type migrationWrite struct {
	path    string
	content []byte
}

// MigrateRecords brings every Job Listing and Application under dataDir to
// CurrentSchemaVersion (issue #100, ADR-0034), backfilling only what is
// mechanically recoverable and reporting what is not.
//
// It reads and plans every record before writing any, so an unparseable
// record, or one stamped at a version this build does not know, halts the
// whole run with an error naming the file and leaves every file untouched.
// Records already at CurrentSchemaVersion are never rewritten, so running
// it again is a no-op. Each rewrite goes through a temporary file and a
// rename. A missing or empty dataDir is not an error.
//
// A Job Listing's Markdown body (its Job Description) is kept byte for
// byte, and keys the migration does not know about are kept too: the
// frontmatter or document is edited as a YAML node tree, never round-tripped
// through the typed record.
func MigrateRecords(dataDir string, opts MigrationOptions) (MigrationReport, error) {
	report := MigrationReport{DryRun: opts.DryRun}

	listingSlugs, err := recordSlugs(filepath.Join(dataDir, jobsDir))
	if err != nil {
		return report, err
	}
	applicationSlugs, err := recordSlugs(filepath.Join(dataDir, applicationsDir))
	if err != nil {
		return report, err
	}
	hasApplication := make(map[string]bool, len(applicationSlugs))
	for _, slug := range applicationSlugs {
		hasApplication[slug] = true
	}

	var writes []migrationWrite
	for _, slug := range listingSlugs {
		rel := path.Join(jobsDir, slug+".md")
		content, err := os.ReadFile(filepath.Join(dataDir, filepath.FromSlash(rel)))
		if err != nil {
			return report, fmt.Errorf("%s: %w", rel, err)
		}
		report.Scanned++

		migrated, change, err := migrateJobListing(content)
		if err != nil {
			return report, fmt.Errorf("%s: %w", rel, err)
		}
		if !hasApplication[slug] {
			report.Inconsistencies = append(report.Inconsistencies, Inconsistency{
				Path:    rel,
				Problem: fmt.Sprintf("no Application file at %s", path.Join(applicationsDir, slug+".md")),
			})
		}
		if change == nil {
			continue
		}
		change.Path = rel
		report.Migrated = append(report.Migrated, *change)
		writes = append(writes, migrationWrite{path: filepath.Join(dataDir, filepath.FromSlash(rel)), content: migrated})
	}

	if opts.DryRun {
		return report, nil
	}
	for _, w := range writes {
		if err := writeFileAtomic(w.path, w.content); err != nil {
			return report, err
		}
	}
	return report, nil
}

// migrateJobListing returns content rewritten at CurrentSchemaVersion and
// what changed, or a nil change when the Job Listing is already current.
func migrateJobListing(content []byte) ([]byte, *RecordMigration, error) {
	fm, closingAndBody, err := locateFrontmatter(content)
	if err != nil {
		return nil, nil, err
	}
	var raw rawJobListingFrontmatter
	if err := yaml.Unmarshal(fm, &raw); err != nil {
		return nil, nil, err
	}
	if err := checkSchemaVersion(raw.SchemaVersion); err != nil {
		return nil, nil, err
	}
	if raw.SchemaVersion == CurrentSchemaVersion {
		return nil, nil, nil
	}

	doc, mapping, err := parseMapping(fm)
	if err != nil {
		return nil, nil, err
	}
	change := &RecordMigration{Kind: RecordJobListing, FromVersion: raw.SchemaVersion, ToVersion: CurrentSchemaVersion}

	// Every reader already treats a missing freshnessStatus as
	// not-yet-checked; writing it down is recovery, not estimation.
	if raw.FreshnessStatus == "" {
		if err := setMappingValue(mapping, "freshnessStatus", FreshnessNotYetChecked); err != nil {
			return nil, nil, err
		}
		change.Backfilled = append(change.Backfilled, BackfilledField{Field: "freshnessStatus", Value: string(FreshnessNotYetChecked)})
	}
	stampSchemaVersion(mapping)

	encoded, err := yaml.Marshal(doc)
	if err != nil {
		return nil, nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(encoded)
	buf.Write(closingAndBody)
	return buf.Bytes(), change, nil
}

// recordSlugs lists the record ids under dir: every regular *.md file,
// skipping dotfiles (a temporary file left by an interrupted write) and
// anything else, such as Company Logo images. A missing dir has none.
func recordSlugs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var slugs []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".md") {
			continue
		}
		slugs = append(slugs, strings.TrimSuffix(name, ".md"))
	}
	return slugs, nil
}

// parseMapping parses a YAML document whose root must be a mapping,
// returning the document node (for re-encoding) and the mapping itself.
func parseMapping(content []byte) (*yaml.Node, *yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("record is not a YAML mapping")
	}
	return &doc, doc.Content[0], nil
}

// stampSchemaVersion sets schemaVersion to CurrentSchemaVersion, as the
// first key so the version is the first thing a reader of the file sees.
func stampSchemaVersion(mapping *yaml.Node) {
	removeMappingKey(mapping, "schemaVersion")
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "schemaVersion"}
	value := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(CurrentSchemaVersion)}
	mapping.Content = append([]*yaml.Node{key, value}, mapping.Content...)
}

// setMappingValue sets key to value's YAML encoding, replacing an existing
// value in place or appending the key at the end.
func setMappingValue(mapping *yaml.Node, key string, value any) error {
	var encoded yaml.Node
	if err := encoded.Encode(value); err != nil {
		return err
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = &encoded
			return nil
		}
	}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &encoded)
	return nil
}

func removeMappingKey(mapping *yaml.Node, key string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

// writeFileAtomic replaces path with content via a temporary file in the
// same directory and a rename, so an interrupted write leaves either the
// old file or the new one, never a truncated one. It keeps the existing
// file's permissions. Narrow on purpose: once issue #88's shared atomic
// write helper lands, this should be replaced by it.
func writeFileAtomic(path string, content []byte) (err error) {
	perm := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		perm = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(content); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Chmod(perm); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
