package note

import "time"

// Note represents a note with metadata and optional content
type Note struct {
	Filename  string    `json:"filename"`
	Path      string    `json:"path"`
	Content   string    `json:"content,omitempty"`
	FirstLine string    `json:"firstLine"`
	ModTime   time.Time `json:"modTime"`
	Labels    []string  `json:"labels,omitempty"`
}

// Frontmatter represents YAML frontmatter in a note
type Frontmatter struct {
	Labels []string `yaml:"labels"`
}

// HasLabel returns true if the note has the specified label
func (n Note) HasLabel(label string) bool {
	for _, l := range n.Labels {
		if l == label {
			return true
		}
	}
	return false
}
