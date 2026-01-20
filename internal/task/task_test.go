package task

import (
	"testing"
)

func TestValidate_EmptyTitle(t *testing.T) {
	task := NewTask(todo, "", "description")
	err := task.Validate()
	if err != ErrEmptyTitle {
		t.Errorf("expected ErrEmptyTitle, got %v", err)
	}
}

func TestValidate_InvalidStatus(t *testing.T) {
	task := Task{
		id:          "test-id",
		status:      status(-1),
		title:       "valid title",
		description: "description",
	}
	err := task.Validate()
	if err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}

	task.status = status(99)
	err = task.Validate()
	if err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus for status 99, got %v", err)
	}
}

func TestValidate_Success(t *testing.T) {
	task := NewTask(todo, "valid title", "description")
	err := task.Validate()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	task = NewTask(inProgress, "another task", "")
	err = task.Validate()
	if err != nil {
		t.Errorf("expected nil error for inProgress task, got %v", err)
	}

	task = NewTask(done, "done task", "with description")
	err = task.Validate()
	if err != nil {
		t.Errorf("expected nil error for done task, got %v", err)
	}
}

func TestStatusGetNext(t *testing.T) {
	tests := []struct {
		current  status
		expected status
	}{
		{todo, inProgress},
		{inProgress, done},
		{done, todo},
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
		current  status
		expected status
	}{
		{todo, done},
		{inProgress, todo},
		{done, inProgress},
	}

	for _, tt := range tests {
		result := tt.current.getPrev()
		if result != tt.expected {
			t.Errorf("getPrev(%d) = %d, expected %d", tt.current, result, tt.expected)
		}
	}
}

func TestSetStatus(t *testing.T) {
	task := NewTask(todo, "test", "desc")

	err := task.SetStatus(inProgress)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if task.Status() != inProgress {
		t.Errorf("expected status inProgress, got %v", task.Status())
	}

	err = task.SetStatus(status(-1))
	if err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}

	err = task.SetStatus(status(99))
	if err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus for status 99, got %v", err)
	}
}
