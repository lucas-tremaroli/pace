package task

import "github.com/google/uuid"

type Task struct {
	id          string
	status      status
	title       string
	description string
}

func NewTask(status status, title, description string) Task {
	return Task{
		id:          uuid.New().String(),
		status:      status,
		title:       title,
		description: description,
	}
}

func NewTaskWithID(id string, status status, title, description string) Task {
	return Task{
		id:          id,
		status:      status,
		title:       title,
		description: description,
	}
}

func (t Task) FilterValue() string {
	return t.title
}

func (t Task) Title() string {
	return t.title
}

func (t Task) Description() string {
	return t.description
}

func (t Task) ID() string {
	return t.id
}

func (t Task) Status() status {
	return t.status
}

// SetStatus updates the task status with validation
func (t *Task) SetStatus(s status) error {
	if s < todo || s > done {
		return ErrInvalidStatus
	}
	t.status = s
	return nil
}

// Validate checks if the task has valid data
func (t Task) Validate() error {
	if t.title == "" {
		return ErrEmptyTitle
	}
	if t.status < todo || t.status > done {
		return ErrInvalidStatus
	}
	return nil
}

type status int

func (s status) getNext() status {
	if s == done {
		return todo
	}
	return s + 1
}

func (s status) getPrev() status {
	if s == todo {
		return done
	}
	return s - 1
}

const (
	todo status = iota
	inProgress
	done
)
