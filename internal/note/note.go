package note

import (
	"slices"
	"time"
)

// Note represents a note with metadata and optional content
type Note struct {
	Filename    string    `json:"filename"`
	Path        string    `json:"path"`
	Content     string    `json:"content,omitempty"`
	Description string    `json:"description"`
	ModTime     time.Time `json:"modTime"`
	Labels      []string  `json:"labels,omitempty"`
	Tasks       []string  `json:"tasks,omitempty"`
}

// Frontmatter represents YAML frontmatter in a note
type Frontmatter struct {
	Description string   `yaml:"description"`
	Labels      []string `yaml:"labels"`
}

// HasLabel returns true if the note has the specified label
func (n Note) HasLabel(label string) bool {
	return slices.Contains(n.Labels, label)
}
