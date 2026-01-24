package task

import (
	"github.com/lucas-tremaroli/pace/internal/storage"
)

// Service handles task business logic and database operations
type Service struct {
	db     *storage.DB
	prefix string
}

// NewService creates a new task service
func NewService() (*Service, error) {
	db, err := storage.NewDB()
	if err != nil {
		return nil, err
	}

	// Initialize or get the ID prefix
	prefix, err := GetOrInitPrefix(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Service{db: db, prefix: prefix}, nil
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

	return s.db.CreateTask(task.ID(), task.Title(), task.Description(), int(task.Status()), int(task.Type()), task.Priority())
}

// UpdateTask updates an existing task in the database
func (s *Service) UpdateTask(task Task) error {
	if err := task.Validate(); err != nil {
		return err
	}

	return s.db.UpdateTask(task.ID(), task.Title(), task.Description(), int(task.Status()), int(task.Type()), task.Priority())
}

// DeleteTask removes a task from the database and cleans up dependencies and labels
func (s *Service) DeleteTask(taskID string) error {
	// Remove all dependencies involving this task first
	if err := s.db.RemoveAllDependencies(taskID); err != nil {
		return err
	}
	// Remove all labels for this task
	if err := s.db.RemoveAllLabels(taskID); err != nil {
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

	var tasks []Task
	for _, record := range taskRecords {
		task := NewTaskComplete(record.ID, Status(record.Status), TaskType(record.TaskType), record.Title, record.Description, record.Priority)
		task.SetBlockedBy(blockedByMap[record.ID])
		task.SetBlocks(blocksMap[record.ID])
		task.SetLabels(labelsMap[record.ID])
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTaskByID retrieves a single task by its ID with dependencies and labels
func (s *Service) GetTaskByID(taskID string) (*Task, error) {
	record, err := s.db.GetTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	task := NewTaskComplete(record.ID, Status(record.Status), TaskType(record.TaskType), record.Title, record.Description, record.Priority)

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

// AddLabel adds a label to a task
func (s *Service) AddLabel(taskID, label string) error {
	// Verify task exists
	if _, err := s.db.GetTaskByID(taskID); err != nil {
		return err
	}
	return s.db.AddLabel(taskID, label)
}

// RemoveLabel removes a label from a task
func (s *Service) RemoveLabel(taskID, label string) error {
	return s.db.RemoveLabel(taskID, label)
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
