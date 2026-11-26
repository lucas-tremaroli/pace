package cmdutil

import (
	"github.com/lucas-tremaroli/pace/v2/internal/epic"
	"github.com/lucas-tremaroli/pace/v2/internal/output"
	"github.com/lucas-tremaroli/pace/v2/internal/task"
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

// WithEpicService initializes an epic service, runs fn, then closes the service.
func WithEpicService(fn func(*epic.Service) error) error {
	svc, err := epic.NewService()
	if err != nil {
		return output.Error(err)
	}
	defer svc.Close()
	return fn(svc)
}
