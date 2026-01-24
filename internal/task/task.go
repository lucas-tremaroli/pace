package task

import (
	"fmt"
)

type Task struct {
	id          string
	status      Status
	title       string
	description string
	priority    int
	blockedBy   []string
	blocks      []string
}

// TaskJSON is the JSON-serializable representation of a Task
type TaskJSON struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	Blocks      []string `json:"blocks,omitempty"`
}

// NewTask creates a new task with the given ID
func NewTask(id string, status Status, title, description string) Task {
	return Task{
		id:          id,
		status:      status,
		title:       title,
		description: description,
		priority:    0,
	}
}

// NewTaskWithID is an alias for NewTask for compatibility
func NewTaskWithID(id string, status Status, title, description string) Task {
	return NewTask(id, status, title, description)
}

// NewTaskFull creates a new task with all fields specified
func NewTaskFull(id string, status Status, title, description string, priority int) Task {
	return Task{
		id:          id,
		status:      status,
		title:       title,
		description: description,
		priority:    priority,
	}
}

// NewTaskWithPriority creates a new task with specified ID and priority
func NewTaskWithPriority(id string, status Status, title, description string, priority int) Task {
	return NewTaskFull(id, status, title, description, priority)
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

// BlockedBy returns the IDs of tasks that block this task
func (t Task) BlockedBy() []string {
	return t.blockedBy
}

// Blocks returns the IDs of tasks that this task blocks
func (t Task) Blocks() []string {
	return t.blocks
}

// SetBlockedBy sets the tasks that block this task
func (t *Task) SetBlockedBy(ids []string) {
	t.blockedBy = ids
}

// SetBlocks sets the tasks that this task blocks
func (t *Task) SetBlocks(ids []string) {
	t.blocks = ids
}

// AddBlockedBy adds a task ID to the blockedBy list
func (t *Task) AddBlockedBy(id string) {
	for _, existing := range t.blockedBy {
		if existing == id {
			return // Already exists
		}
	}
	t.blockedBy = append(t.blockedBy, id)
}

// AddBlocks adds a task ID to the blocks list
func (t *Task) AddBlocks(id string) {
	for _, existing := range t.blocks {
		if existing == id {
			return // Already exists
		}
	}
	t.blocks = append(t.blocks, id)
}

// RemoveBlockedBy removes a task ID from the blockedBy list
func (t *Task) RemoveBlockedBy(id string) {
	for i, existing := range t.blockedBy {
		if existing == id {
			t.blockedBy = append(t.blockedBy[:i], t.blockedBy[i+1:]...)
			return
		}
	}
}

// RemoveBlocks removes a task ID from the blocks list
func (t *Task) RemoveBlocks(id string) {
	for i, existing := range t.blocks {
		if existing == id {
			t.blocks = append(t.blocks[:i], t.blocks[i+1:]...)
			return
		}
	}
}

// IsBlocked returns true if this task has any unresolved blockers
func (t Task) IsBlocked() bool {
	return len(t.blockedBy) > 0
}

// ToJSON converts a Task to its JSON-serializable form
func (t Task) ToJSON() TaskJSON {
	return TaskJSON{
		ID:          t.id,
		Title:       t.title,
		Description: t.description,
		Status:      t.status.String(),
		Priority:    t.priority,
		BlockedBy:   t.blockedBy,
		Blocks:      t.blocks,
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
