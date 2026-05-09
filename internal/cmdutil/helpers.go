package cmdutil

import (
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
)

// WithTaskService initializes a task service, runs fn, then closes the service.
// If initialization fails the process exits via output.Error.
func WithTaskService(fn func(*task.Service) error) error {
	svc, err := task.NewService()
	if err != nil {
		output.Error(err)
	}
	defer svc.Close()
	return fn(svc)
}

// WithNoteService initializes a note service, runs fn, then closes the service.
// If initialization fails the process exits via output.Error.
func WithNoteService(fn func(*note.Service) error) error {
	svc, err := note.NewService()
	if err != nil {
		output.Error(err)
	}
	defer svc.Close()
	return fn(svc)
}
