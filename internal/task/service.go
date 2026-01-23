package task

import (
	"github.com/lucas-tremaroli/pace/internal/storage"
)

// Service handles task business logic and database operations
type Service struct {
	db *storage.DB
}

// NewService creates a new task service
func NewService() (*Service, error) {
	db, err := storage.NewDB()
	if err != nil {
		return nil, err
	}
	return &Service{db: db}, nil
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

	return s.db.CreateTask(task.ID(), task.Title(), task.Description(), int(task.Status()), task.Priority())
}

// UpdateTask updates an existing task in the database
func (s *Service) UpdateTask(task Task) error {
	if err := task.Validate(); err != nil {
		return err
	}

	return s.db.UpdateTask(task.ID(), task.Title(), task.Description(), int(task.Status()), task.Priority())
}

// DeleteTask removes a task from the database
func (s *Service) DeleteTask(taskID string) error {
	return s.db.DeleteTask(taskID)
}

// LoadAllTasks retrieves all tasks from the database
func (s *Service) LoadAllTasks() ([]Task, error) {
	taskRecords, err := s.db.GetAllTasks()
	if err != nil {
		return nil, err
	}

	var tasks []Task
	for _, record := range taskRecords {
		task := NewTaskFull(record.ID, Status(record.Status), record.Title, record.Description, record.Priority)
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTaskByID retrieves a single task by its ID
func (s *Service) GetTaskByID(taskID string) (*Task, error) {
	record, err := s.db.GetTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	task := NewTaskFull(record.ID, Status(record.Status), record.Title, record.Description, record.Priority)
	return &task, nil
}
