package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/muesli/reflow/truncate"
)

var (
	taskNormal         = lipgloss.NewStyle()
	taskInProgressIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	taskDoneIcon       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	taskDoneText       = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Strikethrough(true)
	taskBlockedIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	taskBlockedText    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	taskSelected       = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	taskMeta           = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	taskPriorityHigh   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	taskPriorityMedium = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	taskPriorityLow    = lipgloss.NewStyle().Foreground(lipgloss.Color("146"))
	labelTask          = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	labelBug           = lipgloss.NewStyle().Foreground(lipgloss.Color("167"))
	labelFeature       = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	labelChore         = lipgloss.NewStyle().Foreground(lipgloss.Color("179"))
	labelDocs          = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
)

// taskDelegate renders task items with status-based styling.
type taskDelegate struct{}

func (d taskDelegate) Height() int                             { return 2 }
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
	hasLink := t.Link() != ""

	var iconStyle, textStyle lipgloss.Style
	if selected {
		switch {
		case t.IsBlocked():
			iconStyle = taskSelected.Foreground(lipgloss.Color("196"))
			textStyle = iconStyle.Underline(hasLink)
		case t.Status() == task.InProgress:
			iconStyle = taskSelected.Foreground(lipgloss.Color("226"))
			textStyle = iconStyle.Underline(hasLink)
		case t.Status() == task.Done:
			iconStyle = taskSelected.Foreground(lipgloss.Color("42"))
			textStyle = iconStyle.Strikethrough(true).Underline(hasLink)
		default:
			iconStyle = taskSelected
			textStyle = taskSelected.Underline(hasLink)
		}
	} else {
		switch {
		case t.IsBlocked():
			iconStyle = taskBlockedIcon
			textStyle = taskBlockedText.Underline(hasLink)
		case t.Status() == task.InProgress:
			iconStyle = taskInProgressIcon
			textStyle = taskNormal.Underline(hasLink)
		case t.Status() == task.Done:
			iconStyle = taskDoneIcon
			textStyle = taskDoneText.Underline(hasLink)
		default:
			iconStyle = taskNormal
			textStyle = taskNormal.Underline(hasLink)
		}
	}

	// Line 1: icon + title + priority indicator + label
	// Reserve space for the priority indicator so truncation doesn't push it off-screen.
	maxW := m.Width() - 2
	var suffix string
	var priStyle lipgloss.Style
	switch t.Priority() {
	case 1:
		suffix = " !"
		priStyle = taskPriorityHigh
	case 2:
		suffix = " ~"
		priStyle = taskPriorityMedium
	case 3:
		suffix = ""
		priStyle = taskPriorityLow
	}
	lbl := t.Label()
	if lbl == "" {
		lbl = "task"
	}
	isDone := t.Status() == task.Done
	lblStyle := labelStyle(lbl)
	if isDone {
		lblStyle = lblStyle.Faint(true)
		priStyle = priStyle.Faint(true)
	}
	lblTag := " " + lblStyle.Render("["+lbl+"]")
	line1 := truncateText(iconStyle.Render(prefix+icon+" ")+textStyle.Render(text), maxW)
	line1 += lblTag + priStyle.Render(suffix)
	fmt.Fprint(w, line1)

	// Line 2: ID
	meta := fmt.Sprintf("    %s", t.ID())
	fmt.Fprint(w, "\n"+truncateText(taskMeta.Render(meta), m.Width()))
}

func labelStyle(lbl string) lipgloss.Style {
	switch lbl {
	case "bug":
		return labelBug
	case "feature":
		return labelFeature
	case "chore":
		return labelChore
	case "docs":
		return labelDocs
	default:
		return labelTask
	}
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

func (d noteDelegate) Height() int                             { return 2 }
func (d noteDelegate) Spacing() int                            { return 0 }
func (d noteDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d noteDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ni, ok := item.(NoteItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	prefix := "  "
	style := taskNormal
	if selected {
		prefix = "> "
		style = taskSelected
	}

	// Line 1: filename without .md
	name := strings.TrimSuffix(ni.Note.Filename, ".md")
	fmt.Fprint(w, truncateText(style.Render(prefix+name), m.Width()))
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
