package generation

import (
	"reflect"
	"testing"
)

func TestCheckParsability(t *testing.T) {
	fields := []expectedField{
		{label: "Name", text: "Jane Doe"},
		{label: "Education section header", text: "Education"},
		{label: "Experience section header", text: "Experience"},
		{label: "Experience employer \"Acme Corp\"", text: "Acme Corp"},
		{label: "Projects section header", text: "Projects"},
		{label: "Project \"Widget Tool\"", text: "Widget Tool"},
	}

	tests := []struct {
		name           string
		extractedText  string
		fields         []expectedField
		wantStatus     ParsabilityStatus
		wantMissing    []string
		wantViolations []string
	}{
		{
			name: "happy path: all fields present in order",
			extractedText: "Jane Doe\n" +
				"Education\nMSc Computer Science\n" +
				"Experience\nAcme Corp\nSenior Engineer\n" +
				"Projects\nWidget Tool\n",
			fields:     fields,
			wantStatus: ParsabilityOK,
		},
		{
			name: "missing field: employer absent from extracted text",
			extractedText: "Jane Doe\n" +
				"Education\nMSc Computer Science\n" +
				"Experience\n" +
				"Projects\nWidget Tool\n",
			fields:      fields,
			wantStatus:  ParsabilityWarning,
			wantMissing: []string{"Experience employer \"Acme Corp\""},
		},
		{
			name: "scrambled order: Experience section renders before Education",
			extractedText: "Jane Doe\n" +
				"Experience\nAcme Corp\n" +
				"Education\nMSc Computer Science\n" +
				"Projects\nWidget Tool\n",
			fields:     fields,
			wantStatus: ParsabilityWarning,
			wantViolations: []string{
				"Experience section header appears before Education section header in the extracted text",
				"Experience employer \"Acme Corp\" appears before Education section header in the extracted text",
			},
		},
		{
			name:          "empty extraction: every field missing",
			extractedText: "",
			fields: []expectedField{
				{label: "Name", text: "Jane Doe"},
				{label: "Education section header", text: "Education"},
			},
			wantStatus:  ParsabilityWarning,
			wantMissing: []string{"Name", "Education section header"},
		},
		{
			name:          "no expected fields: trivially ok",
			extractedText: "anything at all",
			fields:        nil,
			wantStatus:    ParsabilityOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkParsability(tt.extractedText, tt.fields)
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", got.Status, tt.wantStatus)
			}
			if !reflect.DeepEqual(got.MissingFields, orNilStrings(tt.wantMissing)) {
				t.Errorf("MissingFields = %v, want %v", got.MissingFields, tt.wantMissing)
			}
			if !reflect.DeepEqual(got.OrderingViolations, orNilStrings(tt.wantViolations)) {
				t.Errorf("OrderingViolations = %v, want %v", got.OrderingViolations, tt.wantViolations)
			}
		})
	}
}

// orNilStrings normalizes an empty-but-non-nil slice to nil, matching
// checkParsability's zero-value var declarations (append on a nil slice
// that never appends stays nil), so table cases can omit the field for
// "expect none" instead of writing []string{}.
func orNilStrings(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

func TestCVExpectedFields(t *testing.T) {
	cv := cvData{
		Name: "Jane Doe",
		Experience: []cvExperience{
			{Employer: "Acme Corp"},
			{Employer: "Acme Corp"}, // same employer twice (e.g. two Client Engagements) — expect one field, not two
			{Employer: "Globex"},
		},
		Projects:  []cvProject{{Name: "Widget Tool"}},
		TechStack: []string{"Go", "React"},
	}

	fields := cvExpectedFields(cv)

	var labels []string
	for _, f := range fields {
		labels = append(labels, f.label)
	}
	want := []string{
		"Name",
		"Education section header",
		"Experience section header",
		"Experience employer \"Acme Corp\"",
		"Experience employer \"Globex\"",
		"Projects section header",
		"Project \"Widget Tool\"",
		"Tech Stack section header",
	}
	if !reflect.DeepEqual(labels, want) {
		t.Errorf("labels = %v, want %v", labels, want)
	}
}

func TestCVExpectedFields_NoProjectsOrTechStack_OmitsThoseHeaders(t *testing.T) {
	cv := cvData{Name: "Jane Doe"}
	fields := cvExpectedFields(cv)
	for _, f := range fields {
		if f.label == "Projects section header" || f.label == "Tech Stack section header" {
			t.Errorf("expected %q to be omitted when empty, got fields %v", f.label, fields)
		}
	}
}

func TestCoverLetterExpectedFields(t *testing.T) {
	cl := coverLetterData{
		Name: "Jane Doe",
		Body: "\n  Dear Hiring Manager,\nI'm excited to apply.\n",
	}
	fields := coverLetterExpectedFields(cl)
	want := []expectedField{
		{label: "Name", text: "Jane Doe"},
		{label: "Body opening", text: "Dear Hiring Manager,"},
	}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("fields = %+v, want %+v", fields, want)
	}
}

func TestFirstNonEmptyLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "leading blank lines skipped", in: "\n\n  first  \nsecond", want: "first"},
		{name: "single line", in: "only", want: "only"},
		{name: "all blank", in: "\n  \n\t\n", want: ""},
		{name: "empty string", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstNonEmptyLine(tt.in); got != tt.want {
				t.Errorf("firstNonEmptyLine(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
