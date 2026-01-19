package tick

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	accentColor = lipgloss.Color("62")
	dimColor    = lipgloss.Color("240")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			MarginBottom(1)

	timerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(accentColor).
			Padding(0, 2)

	completedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("10"))

	containerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 3).
			MarginTop(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			PaddingLeft(3)

	statusStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)
)

type model struct {
	timer          timer.Model
	progress       progress.Model
	keymap         keymap
	help           help.Model
	quitting       bool
	initialTimeout time.Duration
	running        bool
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
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(30),
		progress.WithoutPercentage(),
	)

	return model{
		timer:          timer.NewWithInterval(timeout, time.Millisecond),
		progress:       p,
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
	if m.quitting {
		if m.timer.Timedout() {
			return completedTitleStyle.Render("✓ Focus session complete!") + "\n"
		}
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Focus Timer"))
	b.WriteString("\n")

	// Timer display with formatted time
	remaining := m.timer.Timeout
	mins := int(remaining.Minutes())
	secs := int(remaining.Seconds()) % 60
	timeStr := fmt.Sprintf(" %02d:%02d ", mins, secs)
	b.WriteString(timerStyle.Render(timeStr))
	b.WriteString("\n\n")

	// Progress bar
	elapsed := m.initialTimeout - remaining
	progressPercent := float64(elapsed) / float64(m.initialTimeout)
	b.WriteString(m.progress.ViewAs(progressPercent))
	b.WriteString("\n\n")

	// Status
	status := "Running"
	if !m.running {
		status = "Paused"
	}
	b.WriteString(statusStyle.Render(status))

	content := b.String()

	// Help text outside the container
	help := helpStyle.Render(m.help.View(m.keymap))

	return containerStyle.Render(content) + "\n" + help + "\n"
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
