package task

import (
	"log"

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

	_, err := s.db.CreateTask(task.Title(), task.Description(), int(task.Status()))
	if err != nil {
		log.Printf("Failed to save task to database: %v", err)
		return err
	}
	return nil
}

// UpdateTask updates an existing task in the database
func (s *Service) UpdateTask(task Task) error {
	if err := task.Validate(); err != nil {
		return err
	}

	err := s.db.UpdateTask(task.ID(), task.Title(), task.Description(), int(task.Status()))
	if err != nil {
		log.Printf("Failed to update task in database: %v", err)
		return err
	}
	return nil
}

// DeleteTask removes a task from the database
func (s *Service) DeleteTask(taskID string) error {
	err := s.db.DeleteTask(taskID)
	if err != nil {
		log.Printf("Failed to delete task from database: %v", err)
		return err
	}
	return nil
}

// LoadAllTasks retrieves all tasks from the database
func (s *Service) LoadAllTasks() ([]Task, error) {
	taskRecords, err := s.db.GetAllTasks()
	if err != nil {
		log.Printf("Failed to load tasks from database: %v", err)
		return nil, err
	}

	var tasks []Task
	for _, record := range taskRecords {
		task := NewTaskWithID(record.ID, status(record.Status), record.Title, record.Description)
		tasks = append(tasks, task)
	}

	return tasks, nil
}
