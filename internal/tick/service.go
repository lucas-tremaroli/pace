package tick

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	dimColor = lipgloss.Color("240")

	timerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	statusStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(dimColor)
)

type model struct {
	timer          timer.Model
	keymap         keymap
	help           help.Model
	quitting       bool
	done           bool
	initialTimeout time.Duration
	running        bool
	width          int
	height         int
}

type keymap struct {
	startStop key.Binding
	reset     key.Binding
	quit      key.Binding
}

func (m model) Init() tea.Cmd {
	return m.timer.Init()
}

func NewModel(timeout time.Duration) model {
	return model{
		timer:          timer.NewWithInterval(timeout, time.Millisecond),
		initialTimeout: timeout,
		running:        true,
		keymap: keymap{
			startStop: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "start/stop"),
			),
			reset: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "reset"),
			),
			quit: key.NewBinding(
				key.WithKeys("q"),
				key.WithHelp("q", "quit"),
			),
		},
		help: help.New(),
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.startStop):
			m.running = !m.running
			return m, m.timer.Toggle()
		case key.Matches(msg, m.keymap.reset):
			m.timer = timer.NewWithInterval(m.initialTimeout, time.Millisecond)
			m.running = true
			return m, m.timer.Init()
		case key.Matches(msg, m.keymap.quit):
			m.quitting = true
			return m, tea.Quit
		}
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd
	case timer.TimeoutMsg:
		m.done = true
		return m, nil
	}
	var cmd tea.Cmd
	m.timer, cmd = m.timer.Update(msg)
	return m, cmd
}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.startStop, k.reset, k.quit}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.startStop, k.reset, k.quit},
	}
}

func (m model) View() string {
	var timeDisplay string
	var status string

	if m.done {
		timeDisplay = timerStyle.Render("00:00")
		status = statusStyle.Render("Done")
	} else {
		remaining := m.timer.Timeout
		mins := int(remaining.Minutes())
		secs := int(remaining.Seconds()) % 60
		timeDisplay = timerStyle.Render(fmt.Sprintf("%02d:%02d", mins, secs))

		if m.running {
			status = statusStyle.Render("Running")
		} else {
			status = statusStyle.Render("Paused")
		}
	}

	helpText := helpStyle.Render(m.help.View(m.keymap))

	content := lipgloss.JoinVertical(lipgloss.Center,
		timeDisplay,
		status,
		"",
		helpText,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func NewService(minutes int) *Service {
	return &Service{
		minutes: minutes,
	}
}

type Service struct {
	minutes int
}

func (s *Service) Start() {
	p := tea.NewProgram(
		NewModel(time.Duration(s.minutes)*time.Minute),
		tea.WithAltScreen(),
	)
	p.Run()
}
