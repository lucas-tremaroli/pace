package task

import (
	"testing"
)

func TestParseFilter_Status(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Status
		wantErr bool
	}{
		{"todo", "status=todo", Todo, false},
		{"in-progress", "status=in-progress", InProgress, false},
		{"done", "status=done", Done, false},
		{"invalid status", "status=invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ParseFilter(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFilter(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseFilter(%q) unexpected error: %v", tt.input, err)
				return
			}
			if f.Status == nil {
				t.Errorf("ParseFilter(%q) Status is nil", tt.input)
				return
			}
			if *f.Status != tt.want {
				t.Errorf("ParseFilter(%q) Status = %v, want %v", tt.input, *f.Status, tt.want)
			}
		})
	}
}

func TestParseFilter_Type(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TaskType
		wantErr bool
	}{
		{"task", "type=task", TypeTask, false},
		{"bug", "type=bug", TypeBug, false},
		{"feature", "type=feature", TypeFeature, false},
		{"chore", "type=chore", TypeChore, false},
		{"docs", "type=docs", TypeDocs, false},
		{"invalid type", "type=invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ParseFilter(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFilter(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseFilter(%q) unexpected error: %v", tt.input, err)
				return
			}
			if f.Type == nil {
				t.Errorf("ParseFilter(%q) Type is nil", tt.input)
				return
			}
			if *f.Type != tt.want {
				t.Errorf("ParseFilter(%q) Type = %v, want %v", tt.input, *f.Type, tt.want)
			}
		})
	}
}

func TestParseFilter_Priority(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"priority 1", "priority=1", 1, false},
		{"priority 2", "priority=2", 2, false},
		{"priority 3", "priority=3", 3, false},
		{"priority 4", "priority=4", 4, false},
		{"priority 0", "priority=0", 0, true},
		{"priority 5", "priority=5", 0, true},
		{"priority negative", "priority=-1", 0, true},
		{"priority non-numeric", "priority=high", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ParseFilter(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFilter(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseFilter(%q) unexpected error: %v", tt.input, err)
				return
			}
			if f.Priority == nil {
				t.Errorf("ParseFilter(%q) Priority is nil", tt.input)
				return
			}
			if *f.Priority != tt.want {
				t.Errorf("ParseFilter(%q) Priority = %v, want %v", tt.input, *f.Priority, tt.want)
			}
		})
	}
}

func TestParseFilter_Label(t *testing.T) {
	f, err := ParseFilter("label=sprint-1")
	if err != nil {
		t.Fatalf("ParseFilter(label=sprint-1) unexpected error: %v", err)
	}
	if len(f.Labels) != 1 || f.Labels[0] != "sprint-1" {
		t.Errorf("ParseFilter(label=sprint-1) Labels = %v, want [sprint-1]", f.Labels)
	}
}

func TestParseFilter_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"no equals", "status"},
		{"empty", ""},
		{"unknown key", "unknown=value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFilter(tt.input)
			if err == nil {
				t.Errorf("ParseFilter(%q) expected error, got nil", tt.input)
			}
		})
	}
}

func TestTaskFilter_Matches(t *testing.T) {
	// Create test tasks
	todoTask := NewTaskComplete("t1", Todo, TypeFeature, "Todo Feature", "", 1, "")
	todoTask.AddLabel("sprint-1")
	todoTask.AddLabel("urgent")

	doneTask := NewTaskComplete("t2", Done, TypeBug, "Done Bug", "", 2, "")
	doneTask.AddLabel("sprint-1")

	tests := []struct {
		name   string
		filter *TaskFilter
		task   Task
		want   bool
	}{
		{
			name:   "status match",
			filter: &TaskFilter{Status: ptr(Todo)},
			task:   todoTask,
			want:   true,
		},
		{
			name:   "status no match",
			filter: &TaskFilter{Status: ptr(Done)},
			task:   todoTask,
			want:   false,
		},
		{
			name:   "type match",
			filter: &TaskFilter{Type: ptrType(TypeFeature)},
			task:   todoTask,
			want:   true,
		},
		{
			name:   "type no match",
			filter: &TaskFilter{Type: ptrType(TypeBug)},
			task:   todoTask,
			want:   false,
		},
		{
			name:   "priority match",
			filter: &TaskFilter{Priority: ptrInt(1)},
			task:   todoTask,
			want:   true,
		},
		{
			name:   "priority no match",
			filter: &TaskFilter{Priority: ptrInt(2)},
			task:   todoTask,
			want:   false,
		},
		{
			name:   "single label match",
			filter: &TaskFilter{Labels: []string{"sprint-1"}},
			task:   todoTask,
			want:   true,
		},
		{
			name:   "single label no match",
			filter: &TaskFilter{Labels: []string{"sprint-2"}},
			task:   todoTask,
			want:   false,
		},
		{
			name:   "multiple labels match (AND)",
			filter: &TaskFilter{Labels: []string{"sprint-1", "urgent"}},
			task:   todoTask,
			want:   true,
		},
		{
			name:   "multiple labels partial match (AND fails)",
			filter: &TaskFilter{Labels: []string{"sprint-1", "not-present"}},
			task:   todoTask,
			want:   false,
		},
		{
			name:   "combined filters match",
			filter: &TaskFilter{Status: ptr(Todo), Type: ptrType(TypeFeature), Priority: ptrInt(1)},
			task:   todoTask,
			want:   true,
		},
		{
			name:   "combined filters partial match",
			filter: &TaskFilter{Status: ptr(Todo), Type: ptrType(TypeBug)},
			task:   todoTask,
			want:   false,
		},
		{
			name:   "empty filter matches all",
			filter: &TaskFilter{},
			task:   todoTask,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Matches(tt.task)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeFilters_Success(t *testing.T) {
	statusFilter, _ := ParseFilter("status=todo")
	typeFilter, _ := ParseFilter("type=bug")
	priorityFilter, _ := ParseFilter("priority=1")
	labelFilter1, _ := ParseFilter("label=sprint-1")
	labelFilter2, _ := ParseFilter("label=urgent")

	merged, err := MergeFilters([]*TaskFilter{statusFilter, typeFilter, priorityFilter, labelFilter1, labelFilter2})
	if err != nil {
		t.Fatalf("MergeFilters() unexpected error: %v", err)
	}

	if merged.Status == nil || *merged.Status != Todo {
		t.Errorf("MergeFilters() Status = %v, want Todo", merged.Status)
	}
	if merged.Type == nil || *merged.Type != TypeBug {
		t.Errorf("MergeFilters() Type = %v, want TypeBug", merged.Type)
	}
	if merged.Priority == nil || *merged.Priority != 1 {
		t.Errorf("MergeFilters() Priority = %v, want 1", merged.Priority)
	}
	if len(merged.Labels) != 2 {
		t.Errorf("MergeFilters() Labels = %v, want 2 labels", merged.Labels)
	}
}

func TestMergeFilters_DuplicateStatus(t *testing.T) {
	filter1, _ := ParseFilter("status=todo")
	filter2, _ := ParseFilter("status=done")

	_, err := MergeFilters([]*TaskFilter{filter1, filter2})
	if err == nil {
		t.Error("MergeFilters() expected error for duplicate status, got nil")
	}
}

func TestMergeFilters_DuplicateType(t *testing.T) {
	filter1, _ := ParseFilter("type=bug")
	filter2, _ := ParseFilter("type=feature")

	_, err := MergeFilters([]*TaskFilter{filter1, filter2})
	if err == nil {
		t.Error("MergeFilters() expected error for duplicate type, got nil")
	}
}

func TestMergeFilters_DuplicatePriority(t *testing.T) {
	filter1, _ := ParseFilter("priority=1")
	filter2, _ := ParseFilter("priority=2")

	_, err := MergeFilters([]*TaskFilter{filter1, filter2})
	if err == nil {
		t.Error("MergeFilters() expected error for duplicate priority, got nil")
	}
}

func TestMergeFilters_MultipleLabelsAllowed(t *testing.T) {
	filter1, _ := ParseFilter("label=sprint-1")
	filter2, _ := ParseFilter("label=urgent")
	filter3, _ := ParseFilter("label=backend")

	merged, err := MergeFilters([]*TaskFilter{filter1, filter2, filter3})
	if err != nil {
		t.Fatalf("MergeFilters() unexpected error: %v", err)
	}

	if len(merged.Labels) != 3 {
		t.Errorf("MergeFilters() Labels count = %d, want 3", len(merged.Labels))
	}

	expectedLabels := map[string]bool{"sprint-1": true, "urgent": true, "backend": true}
	for _, label := range merged.Labels {
		if !expectedLabels[label] {
			t.Errorf("MergeFilters() unexpected label: %s", label)
		}
	}
}

// Helper functions for creating pointers
func ptr(s Status) *Status {
	return &s
}

func ptrType(t TaskType) *TaskType {
	return &t
}

func ptrInt(i int) *int {
	return &i
}
