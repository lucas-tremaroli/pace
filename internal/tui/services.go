package tui

import (
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/lucas-tremaroli/pace/internal/task"
)

// TaskService is the narrowed interface the TUI uses to interact with task
// storage. Defined here (not in the task package) so the TUI owns its
// dependencies and tests can substitute lightweight fakes.
type TaskService interface {
	LoadAllTasks() ([]task.Task, error)
	GetTaskByID(id string) (*task.Task, error)
	GetTaskLogs(id string) ([]storage.LogRecord, error)
	GenerateTaskID() string
	CreateTask(t task.Task) error
	UpdateTask(t task.Task) error
	DeleteTask(id string) error
	CloseTask(id, outcome string) error
	SetLabel(id, label string) error
	Close() error
}

// NoteService is the narrowed interface the TUI uses for note storage.
type NoteService interface {
	ListNotesWithMeta(includeContent bool) ([]note.Note, error)
	ReadNoteWithMeta(filename string) (*note.Note, error)
	WriteNote(filename, content string) error
	DeleteNote(filename string) error
	GetNotePath(filename string) string
	Close() error
}
