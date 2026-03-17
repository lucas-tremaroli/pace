package task

import (
	"fmt"
	"strconv"
	"strings"
)

// TaskFilter represents criteria for filtering tasks
type TaskFilter struct {
	Status     *Status
	Type       *TaskType
	Priority   *int   // Single priority, AND semantics (used by bulk ops)
	Priorities []int  // Multiple priorities, OR semantics (used by list)
	Label      *string // Filter by label
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
	case "type":
		taskType, err := ParseTaskType(value)
		if err != nil {
			return nil, err
		}
		filter.Type = &taskType
	case "priority":
		priority, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid priority: %s", value)
		}
		if priority < 1 || priority > 4 {
			return nil, fmt.Errorf("priority must be 1-4, got %d", priority)
		}
		filter.Priority = &priority
	case "label":
		filter.Label = &value
	default:
		return nil, fmt.Errorf("unknown filter key: %s (valid: status, type, priority, label)", key)
	}

	return filter, nil
}

// Matches returns true if the task matches all filter criteria
func (f *TaskFilter) Matches(t Task) bool {
	if f.Status != nil && t.Status() != *f.Status {
		return false
	}
	if f.Type != nil && t.Type() != *f.Type {
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
	return true
}

// Apply returns a filtered copy of tasks, keeping only those that match the filter.
func (f *TaskFilter) Apply(tasks []Task) []Task {
	if f == nil {
		return tasks
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
		if f.Type != nil {
			if merged.Type != nil {
				return nil, fmt.Errorf("duplicate filter: type specified multiple times")
			}
			merged.Type = f.Type
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
	}
	return merged, nil
}
