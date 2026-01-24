package task

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// taskDelegate renders tasks with dependency indicators
type taskDelegate struct {
	baseDelegate list.DefaultDelegate
}

func newTaskDelegate() taskDelegate {
	d := list.NewDefaultDelegate()
	d.SetHeight(1)
	d.ShowDescription = false
	return taskDelegate{baseDelegate: d}
}

func (d taskDelegate) Height() int {
	return d.baseDelegate.Height()
}

func (d taskDelegate) Spacing() int {
	return d.baseDelegate.Spacing()
}

func (d taskDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return d.baseDelegate.Update(msg, m)
}

func (d taskDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	task, ok := item.(Task)
	if !ok {
		d.baseDelegate.Render(w, m, index, item)
		return
	}

	// Build the title with dependency indicators
	title := task.title

	// Add label indicators
	var labelStr string
	for _, label := range task.labels {
		labelStr += fmt.Sprintf(" [%s]", label)
	}

	// Add dependency indicators
	var indicators string
	if len(task.blockedBy) > 0 {
		// Show blocked indicator with count
		indicators += fmt.Sprintf(" [blocked:%d]", len(task.blockedBy))
	}
	if len(task.blocks) > 0 {
		// Show blocking indicator with count
		indicators += fmt.Sprintf(" [blocks:%d]", len(task.blocks))
	}

	// Type prefix
	typePrefix := fmt.Sprintf("[%s] ", task.taskType.Symbol())

	// Styles
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	blockedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	indicatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // Cyan for labels
	typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // Gray for type

	isSelected := index == m.Index()
	isCursor := isSelected

	// Determine if task is blocked (has incomplete blockers)
	isBlocked := len(task.blockedBy) > 0

	var rendered string
	if isCursor {
		cursor := "> "
		if isBlocked {
			rendered = selectedStyle.Render(cursor) + typeStyle.Render(typePrefix) + blockedStyle.Render(title) + labelStyle.Render(labelStr) + indicatorStyle.Render(indicators)
		} else {
			rendered = selectedStyle.Render(cursor) + typeStyle.Render(typePrefix) + selectedStyle.Render(title) + labelStyle.Render(labelStr) + indicatorStyle.Render(indicators)
		}
	} else {
		cursor := "  "
		if isBlocked {
			rendered = cursor + typeStyle.Render(typePrefix) + blockedStyle.Render(title) + labelStyle.Render(labelStr) + indicatorStyle.Render(indicators)
		} else {
			rendered = cursor + typeStyle.Render(typePrefix) + normalStyle.Render(title) + labelStyle.Render(labelStr) + indicatorStyle.Render(indicators)
		}
	}

	fmt.Fprint(w, rendered)
}
