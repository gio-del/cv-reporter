package masterdata

// Entry is a Master Data Entry (Client Engagement or project), covering the
// union of fields either kind of Entry frontmatter can carry. Which fields
// are meaningful is determined by Type.
type Entry struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Employer string   `json:"employer,omitempty"`
	Client   string   `json:"client,omitempty"`
	Role     string   `json:"role,omitempty"`
	Name     string   `json:"name,omitempty"`
	Location string   `json:"location,omitempty"`
	Start    string   `json:"start"`
	End      *string  `json:"end"`
	Flagship bool     `json:"flagship,omitempty"`
	Tags     []string `json:"tags"`
	Repo     string   `json:"repo,omitempty"`
	Bullets  []string `json:"bullets,omitempty"`
}

const (
	TypeExperience = "experience"
	TypeProject    = "project"
)
