package task

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/storage"
)

// Service handles task business logic and database operations
type Service struct {
	db     TaskRepository
	prefix string
}

// NewService creates a new task service backed by the default SQLite storage.
func NewService() (*Service, error) {
	db, err := storage.NewDB()
	if err != nil {
		return nil, err
	}

	prefix, err := GetOrInitPrefix(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Service{db: db, prefix: prefix}, nil
}

// NewServiceWithRepo creates a service with a custom repository and prefix.
// Intended for testing with in-memory implementations.
func NewServiceWithRepo(repo TaskRepository, prefix string) *Service {
	return &Service{db: repo, prefix: prefix}
}

// Prefix returns the current ID prefix
func (s *Service) Prefix() string {
	return s.prefix
}

// GenerateTaskID creates a new unique task ID with the configured prefix
func (s *Service) GenerateTaskID() string {
	return GenerateID(s.prefix)
}

// Close closes the database connection
func (s *Service) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// CreateTask creates a new task and saves it to the database
func (s *Service) CreateTask(task Task) error {
	if err := task.Validate(); err != nil {
		return err
	}

	return s.db.CreateTask(task.ID(), task.Title(), task.Description(), int(task.Status()), task.Priority(), task.Link())
}

// UpdateTask updates an existing task in the database.
// Status changes are rejected for blocked tasks (tasks with unresolved blockers).
func (s *Service) UpdateTask(task Task) error {
	if err := task.Validate(); err != nil {
		return err
	}

	// Reject status changes on blocked tasks
	existing, err := s.db.GetTaskByID(task.ID())
	if err != nil {
		return err
	}
	if Status(existing.Status) != task.Status() {
		blockers, err := s.db.GetBlockers(task.ID())
		if err != nil {
			return err
		}
		if len(blockers) > 0 {
			return fmt.Errorf("cannot change status of blocked task %s (blocked by %s)", task.ID(), strings.Join(blockers, ", "))
		}
	}

	return s.db.UpdateTask(task.ID(), task.Title(), task.Description(), int(task.Status()), task.Priority(), task.Link())
}

// DeleteTask removes a task from the database and cleans up dependencies, labels, and logs
func (s *Service) DeleteTask(taskID string) error {
	// Remove all dependencies involving this task first
	if err := s.db.RemoveAllDependencies(taskID); err != nil {
		return err
	}
	// Remove all labels for this task
	if err := s.db.RemoveAllLabels(taskID); err != nil {
		return err
	}
	// Remove all note links for this task
	if err := s.db.RemoveAllTaskNotes(taskID); err != nil {
		return err
	}
	// Remove all logs for this task
	if err := s.db.DeleteLogsByTaskID(taskID); err != nil {
		return err
	}
	return s.db.DeleteTask(taskID)
}

// LoadAllTasks retrieves all tasks from the database with dependencies and labels
func (s *Service) LoadAllTasks() ([]Task, error) {
	taskRecords, err := s.db.GetAllTasks()
	if err != nil {
		return nil, err
	}

	// Load all dependencies at once for efficiency
	blockedByMap, blocksMap, err := s.db.GetAllDependencies()
	if err != nil {
		return nil, err
	}

	// Load all labels at once for efficiency
	labelsMap, err := s.db.GetAllLabels()
	if err != nil {
		return nil, err
	}

	// Load all note links at once for efficiency
	notesMap, err := s.db.GetAllTaskNotes()
	if err != nil {
		return nil, err
	}

	var tasks []Task
	for _, record := range taskRecords {
		task := NewTaskComplete(record.ID, Status(record.Status), record.Title, record.Description, record.Priority, record.Link)
		task.SetBlockedBy(blockedByMap[record.ID])
		task.SetBlocks(blocksMap[record.ID])
		task.SetLabels(labelsMap[record.ID])
		task.SetNotes(notesMap[record.ID])
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTaskByID retrieves a single task by its ID with dependencies and labels
func (s *Service) GetTaskByID(taskID string) (*Task, error) {
	record, err := s.db.GetTaskByID(taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	task := NewTaskComplete(record.ID, Status(record.Status), record.Title, record.Description, record.Priority, record.Link)

	// Load dependencies for this task
	blockedBy, err := s.db.GetBlockers(taskID)
	if err != nil {
		return nil, err
	}
	blocks, err := s.db.GetBlocking(taskID)
	if err != nil {
		return nil, err
	}
	task.SetBlockedBy(blockedBy)
	task.SetBlocks(blocks)

	// Load labels for this task
	labels, err := s.db.GetLabels(taskID)
	if err != nil {
		return nil, err
	}
	task.SetLabels(labels)

	// Load linked notes for this task
	notes, err := s.db.GetNotesForTask(taskID)
	if err != nil {
		return nil, err
	}
	task.SetNotes(notes)

	return &task, nil
}

// AddDependency creates a blocking relationship where blocker blocks blocked
func (s *Service) AddDependency(blockerID, blockedID string) error {
	// Verify both tasks exist
	if _, err := s.db.GetTaskByID(blockerID); err != nil {
		return err
	}
	if _, err := s.db.GetTaskByID(blockedID); err != nil {
		return err
	}
	return s.db.AddDependency(blockerID, blockedID)
}

// RemoveDependency removes a blocking relationship
func (s *Service) RemoveDependency(blockerID, blockedID string) error {
	return s.db.RemoveDependency(blockerID, blockedID)
}

// SetLabel sets the label for a task, replacing any existing label.
// Pass an empty string to clear the label.
func (s *Service) SetLabel(taskID, label string) error {
	// Verify task exists
	if _, err := s.db.GetTaskByID(taskID); err != nil {
		return err
	}
	return s.db.SetLabel(taskID, label)
}

// LinkNote links a note to a task
func (s *Service) LinkNote(taskID, noteFilename string) error {
	if _, err := s.db.GetTaskByID(taskID); err != nil {
		return err
	}
	return s.db.AddTaskNote(taskID, noteFilename)
}

// UnlinkNote removes a note link from a task
func (s *Service) UnlinkNote(taskID, noteFilename string) error {
	return s.db.RemoveTaskNote(taskID, noteFilename)
}

// GetTasksForNote returns task IDs linked to a note
func (s *Service) GetTasksForNote(noteFilename string) ([]string, error) {
	return s.db.GetTasksForNote(noteFilename)
}

// GetReadyTasks returns tasks that have no blockers or all blockers are done
func (s *Service) GetReadyTasks() ([]Task, error) {
	tasks, err := s.LoadAllTasks()
	if err != nil {
		return nil, err
	}

	// Build a map of task status by ID
	statusMap := make(map[string]Status)
	for _, t := range tasks {
		statusMap[t.ID()] = t.Status()
	}

	var ready []Task
	for _, t := range tasks {
		// Skip completed tasks
		if t.Status() == Done {
			continue
		}

		// Check if all blockers are done
		isReady := true
		for _, blockerID := range t.BlockedBy() {
			if status, exists := statusMap[blockerID]; exists && status != Done {
				isReady = false
				break
			}
		}

		if isReady {
			ready = append(ready, t)
		}
	}

	return ready, nil
}

// LogEntry adds a log entry to a task
func (s *Service) LogEntry(taskID, message string) error {
	if _, err := s.db.GetTaskByID(taskID); err != nil {
		return err
	}
	return s.db.CreateLog(taskID, message, "log")
}

// CloseTask marks a task as done and optionally records an outcome
func (s *Service) CloseTask(taskID, outcome string) error {
	existing, err := s.db.GetTaskByID(taskID)
	if err != nil {
		return err
	}

	if err := s.db.UpdateTask(existing.ID, existing.Title, existing.Description, int(Done), existing.Priority, existing.Link); err != nil {
		return err
	}

	// A completed task can't block anything — clean up outbound deps
	if err := s.db.RemoveBlockingDeps(taskID); err != nil {
		return err
	}

	if outcome != "" {
		return s.db.CreateLog(taskID, outcome, "outcome")
	}
	return nil
}

// GetTaskLogs returns all logs for a task
func (s *Service) GetTaskLogs(taskID string) ([]storage.LogRecord, error) {
	if _, err := s.db.GetTaskByID(taskID); err != nil {
		return nil, err
	}
	return s.db.GetLogsByTaskID(taskID)
}

// SearchLogs performs full-text search across all logs
func (s *Service) SearchLogs(query string, limit int) ([]storage.LogRecord, error) {
	return s.db.SearchLogs(query, limit)
}
