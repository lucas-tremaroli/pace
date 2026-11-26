package output

import (
	"github.com/lucas-tremaroli/pace/v2/internal/task"
)

// TaskListResponse is the standard envelope for listing tasks.
type TaskListResponse struct {
	Tasks []task.TaskJSON `json:"tasks"`
	Count int             `json:"count"`
}
