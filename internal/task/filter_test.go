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
	if f.Label == nil || *f.Label != "sprint-1" {
		t.Errorf("ParseFilter(label=sprint-1) Label = %v, want sprint-1", f.Label)
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
	todoTask := NewTaskComplete("t1", Todo, "Todo Feature", "", 1, "")
	todoTask.SetLabel("feature")

	doneTask := NewTaskComplete("t2", Done, "Done Bug", "", 2, "")
	doneTask.SetLabel("bug")

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
			name:   "label match",
			filter: &TaskFilter{Label: ptrStr("feature")},
			task:   todoTask,
			want:   true,
		},
		{
			name:   "label no match",
			filter: &TaskFilter{Label: ptrStr("bug")},
			task:   todoTask,
			want:   false,
		},
		{
			name:   "combined filters match",
			filter: &TaskFilter{Status: ptr(Todo), Label: ptrStr("feature"), Priority: ptrInt(1)},
			task:   todoTask,
			want:   true,
		},
		{
			name:   "combined filters partial match",
			filter: &TaskFilter{Status: ptr(Todo), Label: ptrStr("bug")},
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
	priorityFilter, _ := ParseFilter("priority=1")
	labelFilter, _ := ParseFilter("label=bug")

	merged, err := MergeFilters([]*TaskFilter{statusFilter, priorityFilter, labelFilter})
	if err != nil {
		t.Fatalf("MergeFilters() unexpected error: %v", err)
	}

	if merged.Status == nil || *merged.Status != Todo {
		t.Errorf("MergeFilters() Status = %v, want Todo", merged.Status)
	}
	if merged.Priority == nil || *merged.Priority != 1 {
		t.Errorf("MergeFilters() Priority = %v, want 1", merged.Priority)
	}
	if merged.Label == nil || *merged.Label != "bug" {
		t.Errorf("MergeFilters() Label = %v, want bug", merged.Label)
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

func TestMergeFilters_DuplicatePriority(t *testing.T) {
	filter1, _ := ParseFilter("priority=1")
	filter2, _ := ParseFilter("priority=2")

	_, err := MergeFilters([]*TaskFilter{filter1, filter2})
	if err == nil {
		t.Error("MergeFilters() expected error for duplicate priority, got nil")
	}
}

func TestMergeFilters_DuplicateLabel(t *testing.T) {
	filter1, _ := ParseFilter("label=sprint-1")
	filter2, _ := ParseFilter("label=urgent")

	_, err := MergeFilters([]*TaskFilter{filter1, filter2})
	if err == nil {
		t.Error("MergeFilters() expected error for duplicate label, got nil")
	}
}

func TestTaskFilter_Matches_Priorities(t *testing.T) {
	task1 := NewTaskComplete("t1", Todo, "Task 1", "", 1, "")
	task2 := NewTaskComplete("t2", Todo, "Task 2", "", 2, "")
	task3 := NewTaskComplete("t3", Todo, "Task 3", "", 3, "")

	tests := []struct {
		name       string
		priorities []int
		task       Task
		want       bool
	}{
		{"matches first priority", []int{1, 2}, task1, true},
		{"matches second priority", []int{1, 2}, task2, true},
		{"no match", []int{1, 2}, task3, false},
		{"empty priorities matches all", []int{}, task3, true},
		{"single match", []int{3}, task3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &TaskFilter{Priorities: tt.priorities}
			got := f.Matches(tt.task)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskFilter_Matches_Label(t *testing.T) {
	task1 := NewTaskComplete("t1", Todo, "Task 1", "", 1, "")
	task1.SetLabel("backend")

	task2 := NewTaskComplete("t2", Todo, "Task 2", "", 1, "")
	task2.SetLabel("frontend")

	task3 := NewTaskComplete("t3", Todo, "Task 3", "", 1, "")

	tests := []struct {
		name  string
		label string
		task  Task
		want  bool
	}{
		{"matches label", "backend", task1, true},
		{"no match", "backend", task2, false},
		{"no label on task", "backend", task3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &TaskFilter{Label: &tt.label}
			got := f.Matches(tt.task)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskFilter_Apply(t *testing.T) {
	task1 := NewTaskComplete("t1", Todo, "Task 1", "", 1, "")
	task2 := NewTaskComplete("t2", Todo, "Task 2", "", 2, "")
	task3 := NewTaskComplete("t3", Done, "Task 3", "", 3, "")
	all := []Task{task1, task2, task3}

	t.Run("nil filter returns all", func(t *testing.T) {
		var f *TaskFilter
		got := f.Apply(all)
		if len(got) != 3 {
			t.Errorf("Apply() len = %d, want 3", len(got))
		}
	})

	t.Run("empty filter returns all", func(t *testing.T) {
		f := &TaskFilter{}
		got := f.Apply(all)
		if len(got) != 3 {
			t.Errorf("Apply() len = %d, want 3", len(got))
		}
	})

	t.Run("status filter", func(t *testing.T) {
		f := &TaskFilter{Status: ptr(Todo)}
		got := f.Apply(all)
		if len(got) != 2 {
			t.Errorf("Apply() len = %d, want 2", len(got))
		}
	})

	t.Run("priorities filter OR semantics", func(t *testing.T) {
		f := &TaskFilter{Priorities: []int{1, 3}}
		got := f.Apply(all)
		if len(got) != 2 {
			t.Errorf("Apply() len = %d, want 2", len(got))
		}
	})

	t.Run("no matches returns empty slice", func(t *testing.T) {
		f := &TaskFilter{Status: ptr(InProgress)}
		got := f.Apply(all)
		if len(got) != 0 {
			t.Errorf("Apply() len = %d, want 0", len(got))
		}
	})
}

// Helper functions for creating pointers
func ptr(s Status) *Status {
	return &s
}

func ptrInt(i int) *int {
	return &i
}

func ptrStr(s string) *string {
	return &s
}
