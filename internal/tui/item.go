package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/muesli/reflow/truncate"
)

var (
	taskNormal      = lipgloss.NewStyle()
	taskInProgressIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	taskDoneIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	taskDoneText    = lipgloss.NewStyle().Faint(true).Strikethrough(true)
	taskBlockedIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	taskBlockedText = lipgloss.NewStyle().Faint(true)
	taskSelected    = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
)

// taskDelegate renders task items with status-based styling.
type taskDelegate struct{}

func (d taskDelegate) Height() int                             { return 1 }
func (d taskDelegate) Spacing() int                            { return 0 }
func (d taskDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d taskDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ti, ok := item.(TaskItem)
	if !ok {
		return
	}

	t := ti.Task
	selected := index == m.Index()
	prefix := "  "
	if selected {
		prefix = "> "
	}

	icon := ti.icon()
	text := t.Title()

	// For in-progress and done tasks, render icon with color separately from text
	if !selected {
		switch {
		case t.IsBlocked():
			line := prefix + taskBlockedIcon.Render(icon+" ") + taskBlockedText.Render(text)
			fmt.Fprint(w, truncateText(line, m.Width()))
			return
		case t.Status() == task.InProgress:
			line := prefix + taskInProgressIcon.Render(icon+" ") + text
			fmt.Fprint(w, truncateText(line, m.Width()))
			return
		case t.Status() == task.Done:
			line := prefix + taskDoneIcon.Render(icon+" ") + taskDoneText.Render(text)
			fmt.Fprint(w, truncateText(line, m.Width()))
			return
		}
	}

	title := prefix + icon + " " + text

	var style lipgloss.Style
	if selected {
		switch {
		case t.IsBlocked():
			style = taskSelected.Foreground(lipgloss.Color("196"))
		case t.Status() == task.InProgress:
			style = taskSelected.Foreground(lipgloss.Color("226"))
		case t.Status() == task.Done:
			iconStyle := taskSelected.Foreground(lipgloss.Color("42"))
			textStyle := iconStyle.Strikethrough(true)
			line := iconStyle.Render(prefix+icon+" ") + textStyle.Render(text)
			fmt.Fprint(w, truncateText(line, m.Width()))
			return
		default:
			style = taskSelected
		}
	} else {
		style = taskNormal
	}

	fmt.Fprint(w, truncateText(style.Render(title), m.Width()))
}

func truncateText(s string, maxW int) string {
	return truncate.StringWithTail(s, uint(maxW), "…")
}

// TaskItem wraps a task for use in a bubbles list.
type TaskItem struct {
	Task task.Task
}

func (i TaskItem) icon() string {
	if i.Task.IsBlocked() {
		return "⊘"
	}
	switch i.Task.Status() {
	case task.InProgress, task.Done:
		return "●"
	default:
		return "○"
	}
}

func (i TaskItem) Title() string {
	return fmt.Sprintf("%s %s", i.icon(), i.Task.Title())
}

func (i TaskItem) Description() string {
	return fmt.Sprintf("%s %s", i.Task.ID(), task.PriorityName(i.Task.Priority()))
}

func (i TaskItem) FilterValue() string {
	return i.Task.Title()
}

// noteDelegate renders note items with consistent styling.
type noteDelegate struct{}

func (d noteDelegate) Height() int                             { return 1 }
func (d noteDelegate) Spacing() int                            { return 0 }
func (d noteDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d noteDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ni, ok := item.(NoteItem)
	if !ok {
		return
	}

	prefix := "  "
	style := taskNormal
	if index == m.Index() {
		prefix = "> "
		style = taskSelected
	}

	fmt.Fprint(w, truncateText(style.Render(prefix+ni.Note.Filename), m.Width()))
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
