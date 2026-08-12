package masterdata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Education, Publication, Award, Activity, and Language are the Static
// Sections carried by profile.yaml alongside contact info (see CONTEXT.md).
type Education struct {
	Degree      string   `yaml:"degree" json:"degree"`
	Institution string   `yaml:"institution" json:"institution"`
	Program     string   `yaml:"program" json:"program"`
	Start       string   `yaml:"start" json:"start"`
	End         string   `yaml:"end" json:"end"`
	Grade       string   `yaml:"grade" json:"grade"`
	Courses     []string `yaml:"courses,omitempty" json:"courses,omitempty"`
}

type Publication struct {
	Title   string `yaml:"title" json:"title"`
	Authors string `yaml:"authors" json:"authors"`
	Venue   string `yaml:"venue" json:"venue"`
	Link    string `yaml:"link,omitempty" json:"link,omitempty"`
	Note    string `yaml:"note,omitempty" json:"note,omitempty"`
}

type Award struct {
	Title       string `yaml:"title" json:"title"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

type Activity struct {
	Title       string `yaml:"title" json:"title"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

type Language struct {
	Name  string `yaml:"name" json:"name"`
	Level string `yaml:"level" json:"level"`
}

// Profile is the contents of profile.yaml: contact info plus the Static
// Sections, always included in full and never subject to Selection or
// Rewrite (see CONTEXT.md).
type Profile struct {
	Name     string `yaml:"name" json:"name"`
	Location string `yaml:"location" json:"location"`
	Email    string `yaml:"email" json:"email"`
	Phone    string `yaml:"phone" json:"phone"`
	LinkedIn string `yaml:"linkedin" json:"linkedin"`
	GitHub   string `yaml:"github" json:"github"`

	Education    []Education   `yaml:"education,omitempty" json:"education"`
	Publications []Publication `yaml:"publications,omitempty" json:"publications"`
	Awards       []Award       `yaml:"awards,omitempty" json:"awards"`
	Activities   []Activity    `yaml:"activities,omitempty" json:"activities"`
	Languages    []Language    `yaml:"languages,omitempty" json:"languages"`
}

// GetProfile reads and parses dataDir/profile.yaml.
func GetProfile(dataDir string) (Profile, error) {
	content, err := os.ReadFile(filepath.Join(dataDir, "profile.yaml"))
	if err != nil {
		return Profile{}, err
	}
	var profile Profile
	if err := yaml.Unmarshal(content, &profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

// UpdateProfile validates profile and, if valid, writes it to
// dataDir/profile.yaml. On validation failure the file is left untouched.
func UpdateProfile(dataDir string, profile Profile) (Profile, error) {
	if err := ValidateProfile(profile); err != nil {
		return Profile{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	content, err := yaml.Marshal(profile)
	if err != nil {
		return Profile{}, err
	}
	if err := os.WriteFile(filepath.Join(dataDir, "profile.yaml"), content, 0o644); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

// ValidateProfile checks that a Profile's required fields are present and
// well-formed before it is written to disk (story 9).
func ValidateProfile(p Profile) error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if p.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.Contains(p.Email, "@") {
		return fmt.Errorf("email must be a valid email address, got %q", p.Email)
	}
	for i, e := range p.Education {
		if e.Degree == "" {
			return fmt.Errorf("education[%d]: degree is required", i)
		}
		if e.Institution == "" {
			return fmt.Errorf("education[%d]: institution is required", i)
		}
	}
	for i, l := range p.Languages {
		if l.Name == "" {
			return fmt.Errorf("languages[%d]: name is required", i)
		}
	}
	return nil
}
