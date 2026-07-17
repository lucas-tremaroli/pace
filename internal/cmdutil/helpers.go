package cmdutil

import (
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
)

// WithTaskService initializes a task service, runs fn, then closes the service.
func WithTaskService(fn func(*task.Service) error) error {
	svc, err := task.NewService()
	if err != nil {
		return output.Error(err)
	}
	defer svc.Close()
	return fn(svc)
}

// WithNoteService initializes a note service, runs fn, then closes the service.
func WithNoteService(fn func(*note.Service) error) error {
	svc, err := note.NewService()
	if err != nil {
		return output.Error(err)
	}
	defer svc.Close()
	return fn(svc)
}

// WithEpicService initializes an epic service, runs fn, then closes the service.
func WithEpicService(fn func(*epic.Service) error) error {
	svc, err := epic.NewService()
	if err != nil {
		return output.Error(err)
	}
	defer svc.Close()
	return fn(svc)
}
