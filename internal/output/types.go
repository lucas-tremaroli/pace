package output

import (
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/task"
)

// TaskListResponse is the standard envelope for listing tasks.
type TaskListResponse struct {
	Tasks []task.TaskJSON `json:"tasks"`
	Count int             `json:"count"`
}

// NoteListResponse is the standard envelope for listing notes.
type NoteListResponse struct {
	Notes []note.Note `json:"notes"`
	Count int         `json:"count"`
}
