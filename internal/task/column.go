package task

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type column struct {
	focus  bool
	status status
	list   list.Model
	height int
	width  int
}

func (c *column) Focus() {
	c.focus = true
}

func (c *column) Blur() {
	c.focus = false
}

func (c *column) Focused() bool {
	return c.focus
}

func newColumn(status status) column {
	var focus bool
	if status == todo {
		focus = true
	}
	defaultList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	defaultList.SetShowHelp(false)
	return column{focus: focus, status: status, list: defaultList}
}

func (c column) Init() tea.Cmd {
	return nil
}

func (c column) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return c.update(msg, nil)
}

func (c column) UpdateWithBoard(msg tea.Msg, board *Board) (tea.Model, tea.Cmd) {
	return c.update(msg, board)
}

func (c column) update(msg tea.Msg, board *Board) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.setSize(msg.Width)
		c.list.SetSize(msg.Width/Margin, msg.Height/2)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Edit):
			if len(c.list.VisibleItems()) != 0 {
				task := c.list.SelectedItem().(Task)
				f := NewForm(task.title, task.description, board)
				f.index = c.list.Index()
				f.col = c
				return f.Update(nil)
			}
		case key.Matches(msg, keys.New):
			f := NewForm("", "", board)
			f.index = AppendIndex
			f.col = c
			return f.Update(nil)
		case key.Matches(msg, keys.Delete):
			return c, c.DeleteCurrent()
		case key.Matches(msg, keys.Enter):
			return c, c.MoveToNext()
		}
	}
	c.list, cmd = c.list.Update(msg)
	return c, cmd
}

func (c column) View() string {
	return c.getStyle().Render(c.list.View())
}

func (c *column) DeleteCurrent() tea.Cmd {
	var task Task
	var ok bool
	if task, ok = c.list.SelectedItem().(Task); !ok {
		return nil
	}

	if len(c.list.VisibleItems()) > 0 {
		c.list.RemoveItem(c.list.Index())
	}

	var cmd tea.Cmd
	c.list, cmd = c.list.Update(nil)
	return tea.Sequence(cmd, func() tea.Msg { return deleteMsg{task} })
}

func (c *column) Set(i int, t Task) tea.Cmd {
	if i != AppendIndex {
		return c.list.SetItem(i, t)
	}
	return c.list.InsertItem(AppendIndex, t)
}

func (c *column) setSize(width int) {
	c.width = width / Margin
}

func (c *column) getStyle() lipgloss.Style {
	if c.Focused() {
		return lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Height(c.height).
			Width(c.width)
	}
	return lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.HiddenBorder()).
		Height(c.height).
		Width(c.width)
}

type moveMsg struct {
	Task
}

type deleteMsg struct {
	Task
}

func (c *column) MoveToNext() tea.Cmd {
	var task Task
	var ok bool
	if task, ok = c.list.SelectedItem().(Task); !ok {
		return nil
	}
	c.list.RemoveItem(c.list.Index())
	task.status = c.status.getNext()

	var cmd tea.Cmd
	c.list, cmd = c.list.Update(nil)

	return tea.Sequence(cmd, func() tea.Msg { return moveMsg{task} })
}
