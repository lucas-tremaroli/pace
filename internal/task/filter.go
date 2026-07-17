package task

import (
	"fmt"
	"strings"
)

// TaskFilter represents criteria for filtering tasks
type TaskFilter struct {
	Status     *Status
	Priority   *int    // Single priority, AND semantics (used by bulk ops)
	Priorities []int   // Multiple priorities, OR semantics (used by list)
	Label      *string // Filter by label
	EpicID     *string // Filter by epic (empty string matches epic-less tasks)
	Ready      bool    // Only show tasks ready to work on (not done, all blockers done)
}

// ParseFilter parses a filter string in the format "key=value"
func ParseFilter(s string) (*TaskFilter, error) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid filter format: %s (expected key=value)", s)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	filter := &TaskFilter{}

	switch key {
	case "status":
		status, err := ParseStatus(value)
		if err != nil {
			return nil, err
		}
		filter.Status = &status
	case "priority":
		priority, err := ParsePriority(value)
		if err != nil {
			return nil, err
		}
		filter.Priority = &priority
	case "label":
		filter.Label = &value
	case "epic":
		filter.EpicID = &value
	default:
		return nil, fmt.Errorf("unknown filter key: %s (valid: status, priority, label, epic)", key)
	}

	return filter, nil
}

// Matches returns true if the task matches all filter criteria
func (f *TaskFilter) Matches(t Task) bool {
	if f.Status != nil && t.Status() != *f.Status {
		return false
	}
	if f.Priority != nil && t.Priority() != *f.Priority {
		return false
	}
	if len(f.Priorities) > 0 {
		found := false
		for _, p := range f.Priorities {
			if t.Priority() == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if f.Label != nil && t.Label() != *f.Label {
		return false
	}
	if f.EpicID != nil && t.EpicID() != *f.EpicID {
		return false
	}
	return true
}

// Apply returns a filtered copy of tasks, keeping only those that match the filter.
func (f *TaskFilter) Apply(tasks []Task) []Task {
	if f == nil {
		return tasks
	}

	// When Ready is set, build a status map and filter for unblocked non-done tasks
	if f.Ready {
		statusMap := make(map[string]Status)
		for _, t := range tasks {
			statusMap[t.ID()] = t.Status()
		}

		var result []Task
		for _, t := range tasks {
			if t.Status() == Done {
				continue
			}
			allBlockersDone := true
			for _, blockerID := range t.BlockedBy() {
				if status, exists := statusMap[blockerID]; exists && status != Done {
					allBlockersDone = false
					break
				}
			}
			if allBlockersDone && f.Matches(t) {
				result = append(result, t)
			}
		}
		return result
	}

	var result []Task
	for _, t := range tasks {
		if f.Matches(t) {
			result = append(result, t)
		}
	}
	return result
}

// MergeFilters combines multiple filters into one that requires all conditions.
// Returns an error if duplicate status, type, priority, or label filters are specified.
func MergeFilters(filters []*TaskFilter) (*TaskFilter, error) {
	merged := &TaskFilter{}
	for _, f := range filters {
		if f.Status != nil {
			if merged.Status != nil {
				return nil, fmt.Errorf("duplicate filter: status specified multiple times")
			}
			merged.Status = f.Status
		}
		if f.Priority != nil {
			if merged.Priority != nil {
				return nil, fmt.Errorf("duplicate filter: priority specified multiple times")
			}
			merged.Priority = f.Priority
		}
		if f.Label != nil {
			if merged.Label != nil {
				return nil, fmt.Errorf("duplicate filter: label specified multiple times")
			}
			merged.Label = f.Label
		}
		if f.EpicID != nil {
			if merged.EpicID != nil {
				return nil, fmt.Errorf("duplicate filter: epic specified multiple times")
			}
			merged.EpicID = f.EpicID
		}
	}
	return merged, nil
}
