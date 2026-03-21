package tui

import (
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/task"
)

// TaskItem wraps a task for use in a bubbles list.
type TaskItem struct {
	Task task.Task
}

func (i TaskItem) Title() string {
	t := i.Task
	var icon string
	if t.IsBlocked() {
		icon = "⊘"
	} else {
		switch t.Status() {
		case task.Todo:
			icon = "○"
		case task.InProgress:
			icon = "●"
		case task.Done:
			icon = "●"
		}
	}
	label := t.Label()
	if label == "" {
		label = "task"
	}
	return fmt.Sprintf("%s %s [%s]", icon, t.Title(), label)
}

func (i TaskItem) Description() string {
	return fmt.Sprintf("%s P%d", i.Task.ID(), i.Task.Priority())
}

func (i TaskItem) FilterValue() string {
	return i.Task.Title()
}

// NoteItem wraps a note for use in a bubbles list.
type NoteItem struct {
	Note note.Note
}

func (i NoteItem) Title() string {
	return i.Note.Filename
}

func (i NoteItem) Description() string {
	return i.Note.Description
}

func (i NoteItem) FilterValue() string {
	return i.Note.Filename
}
