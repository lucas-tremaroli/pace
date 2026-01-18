package tick

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	timer          timer.Model
	keymap         keymap
	help           help.Model
	quitting       bool
	initialTimeout time.Duration
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
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.startStop):
			return m, m.timer.Toggle()
		case key.Matches(msg, m.keymap.reset):
			m.timer = timer.NewWithInterval(m.initialTimeout, time.Millisecond)
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
		m.quitting = true
		return m, tea.Quit
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
	s := "Time left: " + m.timer.View()
	if m.timer.Timedout() {
		s = "Time's up!"
	}
	s += "\n"
	if !m.quitting {
		s += m.help.View(m.keymap)
	}
	return s
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
	p := tea.NewProgram(NewModel(
		time.Duration(s.minutes) * time.Minute,
	))
	p.Run()
}
