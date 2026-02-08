package task

import (
	"fmt"
	"strconv"
	"strings"
)

// TaskFilter represents criteria for filtering tasks
type TaskFilter struct {
	Status   *Status
	Type     *TaskType
	Priority *int
	Label    string
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
		filter.Label = value
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
	if f.Label != "" && !t.HasLabel(f.Label) {
		return false
	}
	return true
}

// MergeFilters combines multiple filters into one that requires all conditions
func MergeFilters(filters []*TaskFilter) *TaskFilter {
	merged := &TaskFilter{}
	for _, f := range filters {
		if f.Status != nil {
			merged.Status = f.Status
		}
		if f.Type != nil {
			merged.Type = f.Type
		}
		if f.Priority != nil {
			merged.Priority = f.Priority
		}
		if f.Label != "" {
			merged.Label = f.Label
		}
	}
	return merged
}

// TaskUpdate represents changes to apply to a task
type TaskUpdate struct {
	Status   *Status
	Type     *TaskType
	Priority *int
}

// ParseSetValue parses a set string in the format "key=value"
func ParseSetValue(s string) (*TaskUpdate, error) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid set format: %s (expected key=value)", s)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	update := &TaskUpdate{}

	switch key {
	case "status":
		status, err := ParseStatus(value)
		if err != nil {
			return nil, err
		}
		update.Status = &status
	case "type":
		taskType, err := ParseTaskType(value)
		if err != nil {
			return nil, err
		}
		update.Type = &taskType
	case "priority":
		priority, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid priority: %s", value)
		}
		if priority < 1 || priority > 4 {
			return nil, fmt.Errorf("priority must be 1-4, got %d", priority)
		}
		update.Priority = &priority
	default:
		return nil, fmt.Errorf("unknown set key: %s (valid: status, type, priority)", key)
	}

	return update, nil
}

// MergeUpdates combines multiple updates into one
func MergeUpdates(updates []*TaskUpdate) *TaskUpdate {
	merged := &TaskUpdate{}
	for _, u := range updates {
		if u.Status != nil {
			merged.Status = u.Status
		}
		if u.Type != nil {
			merged.Type = u.Type
		}
		if u.Priority != nil {
			merged.Priority = u.Priority
		}
	}
	return merged
}
