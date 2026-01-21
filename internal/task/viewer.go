package task

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewerKeyMap struct {
	Back key.Binding
}

func (k viewerKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back}
}

func (k viewerKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Back}}
}

var viewerKeys = viewerKeyMap{
	Back: key.NewBinding(
		key.WithKeys("v", "esc"),
		key.WithHelp("v/esc", "close"),
	),
}

type Viewer struct {
	help   help.Model
	task   Task
	board  *Board
	width  int
	height int
}

func NewViewer(task Task, board *Board) Viewer {
	return Viewer{
		help:  help.New(),
		task:  task,
		board: board,
	}
}

func (v Viewer) Init() tea.Cmd {
	return nil
}

func (v Viewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	case tea.KeyMsg:
		if key.Matches(msg, viewerKeys.Back) {
			if v.board != nil {
				return v.board.Update(nil)
			}
			return v, nil
		}
	}
	return v, nil
}

func (v Viewer) View() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Bold(true).
		MarginBottom(1)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Margin(1, 2)

	helpStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		MarginLeft(2)

	desc := v.task.Description()
	if desc == "" {
		desc = "(no description)"
	}

	titleSection := lipgloss.JoinVertical(
		lipgloss.Left,
		labelStyle.Render("Title:"),
		valueStyle.Render(v.task.Title()),
	)

	descSection := lipgloss.JoinVertical(
		lipgloss.Left,
		labelStyle.Render("Description:"),
		valueStyle.Render(desc),
	)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render("Task Details"),
		titleSection,
		"",
		descSection,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		boxStyle.Render(content),
		helpStyle.Render(v.help.View(viewerKeys)),
	)
}
