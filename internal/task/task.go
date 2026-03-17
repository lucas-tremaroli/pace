package task

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
)

type Task struct {
	id          string
	status      Status
	title       string
	description string
	priority    int
	blockedBy   []string
	blocks      []string
	labels      []string
	notes       []string
	link        string
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
	Label       string   `json:"label,omitempty"`
	Notes       []string `json:"notes,omitempty"`
	Link        string   `json:"link,omitempty"`
}

// TaskInput is used for parsing bulk task creation input
type TaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	Label       string `json:"label"`
	Link        string `json:"link"`
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

// NewTaskComplete creates a new task with all fields
func NewTaskComplete(id string, status Status, title, description string, priority int, link string) Task {
	link = NormalizeLink(link)
	return Task{
		id:          id,
		status:      status,
		title:       title,
		description: description,
		priority:    priority,
		link:        link,
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

func (t Task) Link() string {
	return t.link
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
	if slices.Contains(t.blockedBy, id) {
		return // Already exists
	}
	t.blockedBy = append(t.blockedBy, id)
}

// AddBlocks adds a task ID to the blocks list
func (t *Task) AddBlocks(id string) {
	if slices.Contains(t.blocks, id) {
		return // Already exists
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

// Label returns the task's label (empty string if none)
func (t Task) Label() string {
	if len(t.labels) > 0 {
		return t.labels[0]
	}
	return ""
}

// Labels returns the task's labels (for internal/storage compatibility)
func (t Task) Labels() []string {
	return t.labels
}

// SetLabel sets the task's label (replaces any existing label)
func (t *Task) SetLabel(label string) {
	if label == "" {
		t.labels = nil
	} else {
		t.labels = []string{label}
	}
}

// SetLabels sets the task's labels (used by storage layer loading)
func (t *Task) SetLabels(labels []string) {
	t.labels = labels
}

// Notes returns the task's linked note filenames
func (t Task) Notes() []string {
	return t.notes
}

// SetNotes sets the task's linked note filenames
func (t *Task) SetNotes(notes []string) {
	t.notes = notes
}

// HasLabel returns true if the task has the given label
func (t Task) HasLabel(label string) bool {
	return t.Label() == label
}

// LabelSymbol returns a short symbol for the task's label (for TUI display)
func (t Task) LabelSymbol() string {
	lt, err := ParseTaskType(t.Label())
	if err != nil {
		return "T"
	}
	return lt.Symbol()
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
		Label:       t.Label(),
		Notes:       t.notes,
		Link:        t.link,
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

// NormalizeLink prepends https:// to a link if no scheme is present
func NormalizeLink(link string) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}
	if !strings.Contains(link, "://") {
		link = "https://" + link
	}
	return link
}

// ValidateLink checks if a link is a valid http/https URL
func ValidateLink(link string) error {
	if link == "" {
		return nil
	}
	parsedURL, err := url.Parse(link)
	if err != nil {
		return ErrInvalidLink
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ErrInvalidLink
	}
	if parsedURL.Host == "" {
		return ErrInvalidLink
	}
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
	if err := ValidateLink(t.link); err != nil {
		return err
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

// TaskType represents the type of task
type TaskType int

const (
	TypeTask TaskType = iota
	TypeBug
	TypeFeature
	TypeChore
	TypeDocs
)

// String returns the string representation of a task type
func (t TaskType) String() string {
	switch t {
	case TypeTask:
		return "task"
	case TypeBug:
		return "bug"
	case TypeFeature:
		return "feature"
	case TypeChore:
		return "chore"
	case TypeDocs:
		return "docs"
	default:
		return "task"
	}
}

// Symbol returns a short symbol for the task type (for TUI display)
func (t TaskType) Symbol() string {
	switch t {
	case TypeBug:
		return "B"
	case TypeFeature:
		return "F"
	case TypeChore:
		return "C"
	case TypeDocs:
		return "D"
	default:
		return "T"
	}
}

// ParseTaskType parses a string into a task type value
func ParseTaskType(s string) (TaskType, error) {
	switch s {
	case "task", "":
		return TypeTask, nil
	case "bug":
		return TypeBug, nil
	case "feature":
		return TypeFeature, nil
	case "chore":
		return TypeChore, nil
	case "docs":
		return TypeDocs, nil
	default:
		return TypeTask, fmt.Errorf("invalid label: %s (valid: task, bug, feature, chore, docs)", s)
	}
}

// ValidLabel returns the label strings that are valid task labels
var ValidLabels = []string{"task", "bug", "feature", "chore", "docs"}

// ValidateLabel checks if a label string is valid
func ValidateLabel(s string) error {
	_, err := ParseTaskType(s)
	return err
}
