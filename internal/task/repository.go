package task

import "github.com/lucas-tremaroli/pace/internal/storage"

// TaskRepository abstracts storage operations needed by the task service.
// *storage.DB satisfies this interface; alternative implementations (e.g.
// in-memory) can be used in tests.
type TaskRepository interface {
	Close() error
	// Tasks
	CreateTask(id, title, description string, status, priority int, link string) error
	GetAllTasks() ([]storage.TaskRecord, error)
	GetTaskByID(id string) (*storage.TaskRecord, error)
	UpdateTask(id, title, description string, status, priority int, link string) error
	DeleteTask(id string) error
	// Dependencies
	AddDependency(blockerID, blockedID string) error
	RemoveDependency(blockerID, blockedID string) error
	GetBlockers(taskID string) ([]string, error)
	GetBlocking(taskID string) ([]string, error)
	GetAllDependencies() (map[string][]string, map[string][]string, error)
	RemoveAllDependencies(taskID string) error
	RemoveBlockingDeps(taskID string) error
	// Labels
	SetLabel(taskID, label string) error
	GetLabels(taskID string) ([]string, error)
	GetAllLabels() (map[string][]string, error)
	RemoveAllLabels(taskID string) error
	// Task-note links
	AddTaskNote(taskID, noteFilename string) error
	RemoveTaskNote(taskID, noteFilename string) error
	GetNotesForTask(taskID string) ([]string, error)
	GetTasksForNote(noteFilename string) ([]string, error)
	GetAllTaskNotes() (map[string][]string, error)
	RemoveAllTaskNotes(taskID string) error
	// Logs
	CreateLog(taskID, message, logType string) error
	GetLogsByTaskID(taskID string) ([]storage.LogRecord, error)
	SearchLogs(query string, limit int) ([]storage.LogRecord, error)
	DeleteLogsByTaskID(taskID string) error
	// Config (used for ID prefix management)
	GetConfig(key string) (string, error)
	SetConfig(key, value string) error
}
