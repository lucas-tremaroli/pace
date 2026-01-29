package task

import (
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type column struct {
	focus  bool
	status Status
	list   list.Model
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

func newColumn(status Status) column {
	var focus bool
	if status == Todo {
		focus = true
	}
	delegate := newTaskDelegate()
	defaultList := list.New([]list.Item{}, delegate, 0, 0)
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
				f := NewFormWithTask(task, board)
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
		case key.Matches(msg, keys.View):
			if len(c.list.VisibleItems()) != 0 {
				task := c.list.SelectedItem().(Task)
				return NewViewer(task, board), nil
			}
		case key.Matches(msg, keys.Open):
			if len(c.list.VisibleItems()) != 0 {
				task := c.list.SelectedItem().(Task)
				if task.Link() != "" {
					return c, c.OpenLink(task.Link())
				}
			}
			return c, nil
		case key.Matches(msg, keys.Enter):
			return c, c.MoveToNext()
		}
	}
	c.list, cmd = c.list.Update(msg)
	return c, cmd
}

func (c column) View() string {
	c.list.SetShowStatusBar(len(c.list.Items()) > 0)
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
			Width(c.width)
	}
	return lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("236")).
		Width(c.width)
}

type moveMsg struct {
	Task
}

type deleteMsg struct {
	Task
}

func (c *column) OpenLink(link string) tea.Cmd {
	return func() tea.Msg {
		// Validate and sanitize the link
		link = strings.TrimSpace(link)
		if link == "" {
			return nil
		}

		// Parse and validate the URL
		parsedURL, err := url.Parse(link)
		if err != nil {
			return nil // Invalid URL format
		}

		// Require an explicit scheme - no assumptions
		if parsedURL.Scheme == "" {
			return nil
		}

		// Only allow http and http/https protocol for security
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return nil
		}

		// Require a valid host
		if parsedURL.Host == "" {
			return nil
		}

		// Use OS-specific command to open URL in default browser
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", link)
		case "linux":
			cmd = exec.Command("xdg-open", link)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", link)
		default:
			// Fallback: try xdg-open (common on Unix-like systems)
			cmd = exec.Command("xdg-open", link)
		}
		_ = cmd.Start()
		return nil
	}
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
