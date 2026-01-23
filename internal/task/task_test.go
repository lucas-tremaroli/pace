package task

import (
	"testing"
)

func TestValidate_EmptyTitle(t *testing.T) {
	task := NewTask(Todo, "", "description")
	err := task.Validate()
	if err != ErrEmptyTitle {
		t.Errorf("expected ErrEmptyTitle, got %v", err)
	}
}

func TestValidate_InvalidStatus(t *testing.T) {
	task := Task{
		id:          "test-id",
		status:      Status(-1),
		title:       "valid title",
		description: "description",
	}
	err := task.Validate()
	if err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}

	task.status = Status(99)
	err = task.Validate()
	if err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus for status 99, got %v", err)
	}
}

func TestValidate_Success(t *testing.T) {
	task := NewTask(Todo, "valid title", "description")
	err := task.Validate()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	task = NewTask(InProgress, "another task", "")
	err = task.Validate()
	if err != nil {
		t.Errorf("expected nil error for InProgress task, got %v", err)
	}

	task = NewTask(Done, "done task", "with description")
	err = task.Validate()
	if err != nil {
		t.Errorf("expected nil error for Done task, got %v", err)
	}
}

func TestStatusGetNext(t *testing.T) {
	tests := []struct {
		current  Status
		expected Status
	}{
		{Todo, InProgress},
		{InProgress, Done},
		{Done, Todo},
	}

	for _, tt := range tests {
		result := tt.current.getNext()
		if result != tt.expected {
			t.Errorf("getNext(%d) = %d, expected %d", tt.current, result, tt.expected)
		}
	}
}

func TestStatusGetPrev(t *testing.T) {
	tests := []struct {
		current  Status
		expected Status
	}{
		{Todo, Done},
		{InProgress, Todo},
		{Done, InProgress},
	}

	for _, tt := range tests {
		result := tt.current.getPrev()
		if result != tt.expected {
			t.Errorf("getPrev(%d) = %d, expected %d", tt.current, result, tt.expected)
		}
	}
}

func TestSetStatus(t *testing.T) {
	task := NewTask(Todo, "test", "desc")

	err := task.SetStatus(InProgress)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if task.Status() != InProgress {
		t.Errorf("expected status InProgress, got %v", task.Status())
	}

	err = task.SetStatus(Status(-1))
	if err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}

	err = task.SetStatus(Status(99))
	if err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus for status 99, got %v", err)
	}
}
