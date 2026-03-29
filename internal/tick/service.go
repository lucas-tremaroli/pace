package tick

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	dimColor      = lipgloss.Color("240")
	runningColor  = lipgloss.Color("15")
	pausedColor   = lipgloss.Color("#FF6B6B")
	doneColor     = lipgloss.Color("42")
	resetColor    = lipgloss.Color("#FFEAA7")
	overtimeColor = lipgloss.Color("#FFEAA7")

	progressFilled = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	progressEmpty  = lipgloss.NewStyle().Foreground(lipgloss.Color("236"))

	timerBase = lipgloss.NewStyle().Bold(true)
	helpStyle = lipgloss.NewStyle().Foreground(dimColor)
	goalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Italic(true)
)

type model struct {
	timer          timer.Model
	overtime       stopwatch.Model
	keymap         keymap
	help           help.Model
	quitting       bool
	done           bool
	initialTimeout time.Duration
	running        bool
	width          int
	height         int
	goal           string
	flashing       bool
	showElapsed    bool
	pausedTotal    time.Duration
	lastPauseAt    time.Time
}

type keymap struct {
	startStop key.Binding
	reset     key.Binding
	toggle    key.Binding
	quit      key.Binding
}

type flashDoneMsg struct{}
type bellMsg struct{}

func bellCmd() tea.Msg {
	return bellMsg{}
}

func (m model) Init() tea.Cmd {
	return m.timer.Init()
}

func NewModel(timeout time.Duration, goal string) model {
	return model{
		timer:          timer.NewWithInterval(timeout, time.Millisecond),
		overtime:       stopwatch.NewWithInterval(time.Millisecond),
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
			toggle: key.NewBinding(
				key.WithKeys("t"),
				key.WithHelp("t", "elapsed/remaining"),
			),
			quit: key.NewBinding(
				key.WithKeys("q"),
				key.WithHelp("q", "quit"),
			),
		},
		help: help.New(),
		goal: goal,
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
			if !m.running {
				m.lastPauseAt = time.Now()
			} else if !m.lastPauseAt.IsZero() {
				m.pausedTotal += time.Since(m.lastPauseAt)
				m.lastPauseAt = time.Time{}
			}
			if m.done {
				return m, m.overtime.Toggle()
			}
			return m, m.timer.Toggle()
		case key.Matches(msg, m.keymap.reset):
			m.timer = timer.NewWithInterval(m.initialTimeout, time.Millisecond)
			m.overtime = stopwatch.NewWithInterval(time.Millisecond)
			m.running = true
			m.done = false
			m.flashing = true
			m.showElapsed = false
			m.pausedTotal = 0
			m.lastPauseAt = time.Time{}
			return m, tea.Batch(m.timer.Init(), tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
				return flashDoneMsg{}
			}))
		case key.Matches(msg, m.keymap.toggle):
			if !m.done {
				m.showElapsed = !m.showElapsed
			}
			return m, nil
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
		m.running = true
		return m, tea.Batch(m.overtime.Init(), bellCmd)
	case bellMsg:
		fmt.Print("\a")
		return m, nil
	case stopwatch.TickMsg:
		var cmd tea.Cmd
		m.overtime, cmd = m.overtime.Update(msg)
		return m, cmd
	case stopwatch.StartStopMsg:
		var cmd tea.Cmd
		m.overtime, cmd = m.overtime.Update(msg)
		return m, cmd
	case stopwatch.ResetMsg:
		var cmd tea.Cmd
		m.overtime, cmd = m.overtime.Update(msg)
		return m, cmd
	case flashDoneMsg:
		m.flashing = false
		return m, nil
	}
	var cmd tea.Cmd
	m.timer, cmd = m.timer.Update(msg)
	return m, cmd
}

func (m model) helpBindings() keymap {
	km := m.keymap
	if m.done {
		km.toggle.SetEnabled(false)
	}
	return km
}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.startStop, k.reset, k.toggle, k.quit}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.startStop, k.reset, k.toggle, k.quit},
	}
}

func formatMmSs(d time.Duration) string {
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

func formatDuration(d time.Duration) string {
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", mins, secs)
}

func (m model) timerColor() lipgloss.Color {
	if !m.running {
		return pausedColor
	}
	if m.done {
		if m.overtime.Elapsed() >= time.Second {
			return overtimeColor
		}
		return doneColor
	}
	if m.flashing {
		return resetColor
	}
	return runningColor
}

func (m model) View() string {
	color := m.timerColor()

	var timeStr string
	if m.done {
		ot := m.overtime.Elapsed()
		if ot < time.Second {
			timeStr = "00:00"
		} else {
			timeStr = "+" + formatMmSs(ot)
		}
	} else if m.showElapsed {
		timeStr = formatMmSs(m.initialTimeout - m.timer.Timeout)
	} else {
		timeStr = formatMmSs(m.timer.Timeout)
	}

	barW := 30
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)
	durationLabel := dimStyle.Render(fmt.Sprintf("%dm", int(m.initialTimeout.Minutes())))
	timeDisplay := timerBase.Foreground(color).Render(timeStr)

	// Progress bar
	elapsed := m.initialTimeout - m.timer.Timeout
	var filled int
	if m.done {
		filled = barW
	} else {
		filled = int(float64(elapsed) / float64(m.initialTimeout) * float64(barW))
	}
	filledStyle := progressFilled
	if !m.running {
		filledStyle = filledStyle.Foreground(pausedColor)
	} else if m.done && m.overtime.Elapsed() >= time.Second {
		filledStyle = filledStyle.Foreground(overtimeColor)
	}
	bar := filledStyle.Render(strings.Repeat("━", filled)) +
		progressEmpty.Render(strings.Repeat("─", barW-filled))

	helpText := helpStyle.Render(m.help.View(m.helpBindings()))

	parts := []string{durationLabel, "", timeDisplay, bar}
	if m.goal != "" {
		parts = append(parts, "", goalStyle.Render(m.goal))
	}
	parts = append(parts, "", helpText)

	content := lipgloss.JoinVertical(lipgloss.Center, parts...)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m model) Summary() string {
	// Focused = how far the timer progressed + overtime
	focused := m.initialTimeout - m.timer.Timeout
	if m.done {
		focused = m.initialTimeout
	}
	overtime := m.overtime.Elapsed()
	focused += overtime

	// Paused = accumulated pause time (include current pause if still paused)
	paused := m.pausedTotal
	if !m.lastPauseAt.IsZero() {
		paused += time.Since(m.lastPauseAt)
	}

	summaryStyle := lipgloss.NewStyle().Foreground(doneColor)
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)

	summary := summaryStyle.Render(fmt.Sprintf("Focused for %s", formatDuration(focused)))

	if overtime >= time.Second {
		summary += summaryStyle.Render(fmt.Sprintf(" (+%s overtime)", formatDuration(overtime)))
	}

	if paused >= time.Second {
		summary += dimStyle.Render(fmt.Sprintf(" · paused %s", formatDuration(paused)))
	}

	if m.goal != "" {
		summary += dimStyle.Render(fmt.Sprintf(" — %s", m.goal))
	}

	return summary
}

func NewService(minutes int, goal string) *Service {
	return &Service{
		minutes: minutes,
		goal:    goal,
	}
}

type Service struct {
	minutes int
	goal    string
}

func (s *Service) Start() {
	m := NewModel(time.Duration(s.minutes)*time.Minute, s.goal)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, _ := p.Run()
	if fm, ok := finalModel.(model); ok {
		fmt.Println(fm.Summary())
	}
}
