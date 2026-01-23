package task

import (
	"fmt"

	"github.com/google/uuid"
)

type Task struct {
	id          string
	status      Status
	title       string
	description string
	priority    int
}

// TaskJSON is the JSON-serializable representation of a Task
type TaskJSON struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
}

func NewTask(status Status, title, description string) Task {
	return Task{
		id:          uuid.New().String(),
		status:      status,
		title:       title,
		description: description,
		priority:    0,
	}
}

func NewTaskWithID(id string, status Status, title, description string) Task {
	return Task{
		id:          id,
		status:      status,
		title:       title,
		description: description,
		priority:    0,
	}
}

func NewTaskFull(id string, status Status, title, description string, priority int) Task {
	return Task{
		id:          id,
		status:      status,
		title:       title,
		description: description,
		priority:    priority,
	}
}

func NewTaskWithPriority(status Status, title, description string, priority int) Task {
	return Task{
		id:          uuid.New().String(),
		status:      status,
		title:       title,
		description: description,
		priority:    priority,
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

func (t Task) Status() Status {
	return t.status
}

func (t Task) Priority() int {
	return t.priority
}

// ToJSON converts a Task to its JSON-serializable form
func (t Task) ToJSON() TaskJSON {
	return TaskJSON{
		ID:          t.id,
		Title:       t.title,
		Description: t.description,
		Status:      t.status.String(),
		Priority:    t.priority,
	}
}

// SetStatus updates the task status with validation
func (t *Task) SetStatus(s Status) error {
	if s < Todo || s > Done {
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
	if t.status < Todo || t.status > Done {
		return ErrInvalidStatus
	}
	return nil
}

type Status int

func (s Status) getNext() Status {
	if s == Done {
		return Todo
	}
	return s + 1
}

func (s Status) getPrev() Status {
	if s == Todo {
		return Done
	}
	return s - 1
}

const (
	Todo Status = iota
	InProgress
	Done
)

// String returns the string representation of a status
func (s Status) String() string {
	switch s {
	case Todo:
		return "todo"
	case InProgress:
		return "in-progress"
	case Done:
		return "done"
	default:
		return "unknown"
	}
}

// ParseStatus parses a string into a status value
func ParseStatus(s string) (Status, error) {
	switch s {
	case "todo":
		return Todo, nil
	case "in-progress":
		return InProgress, nil
	case "done":
		return Done, nil
	default:
		return 0, fmt.Errorf("invalid status: %s (valid: todo, in-progress, done)", s)
	}
}
