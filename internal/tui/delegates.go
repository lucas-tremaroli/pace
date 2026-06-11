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
)

// taskItem adapts task.Task to list.Item.
type taskItem struct{ Task task.Task }

func (i taskItem) FilterValue() string { return i.Task.Title() }

// taskDelegate renders task rows with rich styling: icon + title +
// priority suffix + label tag + ID meta line. Two lines tall.
type taskDelegate struct{}

func (taskDelegate) Height() int                             { return 2 }
func (taskDelegate) Spacing() int                            { return 0 }
func (taskDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (taskDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ti, ok := item.(taskItem)
	if !ok {
		return
	}
	t := ti.Task
	selected := index == m.Index()
	prefix := "  "
	if selected {
		prefix = "> "
	}

	text := t.Title()
	hasLink := t.Link() != ""

	var iconStyle, textStyle lipgloss.Style
	if selected {
		iconStyle = theme.TaskSelected
		textStyle = theme.TaskSelected.Underline(hasLink)
	} else {
		switch {
		case t.IsBlocked():
			iconStyle = theme.TaskBlockedIcon
			textStyle = theme.TaskBlockedText.Underline(hasLink)
		case t.Status() == task.InProgress:
			iconStyle = theme.TaskInProgressIcon
			textStyle = theme.TaskNormal.Underline(hasLink)
		case t.Status() == task.Done:
			iconStyle = theme.TaskDoneIcon
			textStyle = theme.TaskDoneText.Underline(hasLink)
		default:
			iconStyle = theme.TaskNormal
			textStyle = theme.TaskNormal.Underline(hasLink)
		}
	}

	maxW := m.Width() - 2
	var suffix string
	var priStyle lipgloss.Style
	switch t.Priority() {
	case 1:
		suffix = " !"
		priStyle = theme.ItemPriorityHigh
	case 2:
		suffix = " ~"
		priStyle = theme.ItemPriorityMedium
	case 3:
		suffix = ""
		priStyle = theme.ItemPriorityLow
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
	var lblTag string
	if lbl != "task" {
		lblTag = " " + lblStyle.Render("["+lbl+"]")
	}
	line1 := truncateText(iconStyle.Render(prefix)+textStyle.Render(text), maxW)
	line1 += lblTag + priStyle.Render(suffix)
	fmt.Fprint(w, line1)

	meta := fmt.Sprintf("  %s", t.ID())
	fmt.Fprint(w, "\n"+truncateText(theme.TaskMeta.Render(meta), m.Width()))
}

// noteItem adapts note.Note to list.Item.
type noteItem struct{ Note note.Note }

func (i noteItem) FilterValue() string { return i.Note.Filename }

// noteDelegate renders one note per row: filename (without .md) and
// description on a dim second line if present.
type noteDelegate struct{}

func (noteDelegate) Height() int                             { return 2 }
func (noteDelegate) Spacing() int                            { return 0 }
func (noteDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (noteDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ni, ok := item.(noteItem)
	if !ok {
		return
	}
	selected := index == m.Index()
	prefix := "  "
	style := theme.TaskNormal
	if selected {
		prefix = "> "
		style = theme.TaskSelected
	}
	name := strings.TrimSuffix(ni.Note.Filename, ".md")
	fmt.Fprint(w, truncateText(style.Render(prefix+name), m.Width()))
	fmt.Fprint(w, "\n")
	desc := ni.Note.Description
	if desc == "" {
		desc = "—"
	}
	fmt.Fprint(w, truncateText(theme.TaskMeta.Render("  "+desc), m.Width()))
}
