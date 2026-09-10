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

	// LastModified is populated by the API layer (via EntryLastModified),
	// never by ListEntries/GetEntry themselves — it depends on a
	// projectRoot they don't take. Left nil when there's no git history to
	// report (a fresh, uncommitted Entry, or a lookup with no repository
	// behind it).
	LastModified *LastModified `json:"lastModified,omitempty"`
}

const (
	TypeExperience = "experience"
	TypeProject    = "project"
)
