package note

import (
	"fmt"
	"strings"
)

// NoteFilter represents criteria for filtering notes
type NoteFilter struct {
	Labels []string // Multiple labels use AND semantics (note must have all)
}

// ParseFilter parses a filter string in the format "key=value"
func ParseFilter(s string) (*NoteFilter, error) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid filter format: %s (expected key=value)", s)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	filter := &NoteFilter{}

	switch key {
	case "label":
		filter.Labels = []string{value}
	default:
		return nil, fmt.Errorf("unknown filter key: %s (valid: label)", key)
	}

	return filter, nil
}

// Matches returns true if the note matches all filter criteria
func (f *NoteFilter) Matches(n Note) bool {
	// All specified labels must be present (AND semantics)
	for _, label := range f.Labels {
		if !n.HasLabel(label) {
			return false
		}
	}
	return true
}

// MergeFilters combines multiple filters into one that requires all conditions.
// Multiple label filters are allowed and use AND semantics (note must have all labels).
func MergeFilters(filters []*NoteFilter) *NoteFilter {
	merged := &NoteFilter{}
	for _, f := range filters {
		merged.Labels = append(merged.Labels, f.Labels...)
	}
	return merged
}
