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
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	dimColor      = lipgloss.Color("240")
	runningColor  = lipgloss.Color("15")
	pausedColor   = lipgloss.Color("#FF6B6B")
	doneColor     = lipgloss.Color("42")
	resetColor    = lipgloss.Color("#FFEAA7")
	overtimeColor = lipgloss.Color("#FFEAA7")

	progressFilled = lipgloss.NewStyle().Foreground(doneColor)
	progressEmpty  = lipgloss.NewStyle().Foreground(lipgloss.Color("236"))

	timerBase = lipgloss.NewStyle().Bold(true)
	dimStyle  = lipgloss.NewStyle().Foreground(dimColor)
	helpStyle = dimStyle
	goalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Italic(true)
	taskStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	setupBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)
	setupTitle = lipgloss.NewStyle().Bold(true).Foreground(doneColor)
	setupMeta  = lipgloss.NewStyle().Foreground(dimColor)
)

const setupFormWidth = 56

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
	task           string
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

// timerExitMsg is dispatched by the timer in place of tea.Quit so the
// rootModel can decide whether to show the post-session review prompt.
type timerExitMsg struct{}

func (m model) Init() tea.Cmd {
	return m.timer.Init()
}

func newTimerModel(timeout time.Duration, goal, task string) model {
	return model{
		timer:          timer.NewWithInterval(timeout, time.Millisecond),
		overtime:       stopwatch.NewWithInterval(time.Millisecond),
		initialTimeout: timeout,
		running:        true,
		task:           task,
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
			return m, func() tea.Msg { return timerExitMsg{} }
		}
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd
	case timer.TimeoutMsg:
		m.done = true
		m.running = true
		return m, tea.Batch(m.overtime.Init(), func() tea.Msg { return bellMsg{} })
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

// focused returns the total time spent in the session: timer progress
// (full duration when done) plus any overtime.
func (m model) focused() time.Duration {
	d := m.initialTimeout - m.timer.Timeout
	if m.done {
		d = m.initialTimeout
	}
	return d + m.overtime.Elapsed()
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
	if m.task != "" {
		parts = append(parts, "", taskStyle.Render("→ "+m.task))
	}
	parts = append(parts, "", helpText)

	content := lipgloss.JoinVertical(lipgloss.Center, parts...)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m model) Summary() string {
	focused := m.focused()
	overtime := m.overtime.Elapsed()

	// Paused = accumulated pause time (include current pause if still paused)
	paused := m.pausedTotal
	if !m.lastPauseAt.IsZero() {
		paused += time.Since(m.lastPauseAt)
	}

	summaryStyle := lipgloss.NewStyle().Foreground(doneColor)
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

	if m.task != "" {
		summary += dimStyle.Render(fmt.Sprintf(" · on %s", m.task))
	}

	return summary
}

// TaskOption is a minimal task descriptor the setup form can render
// without dragging the full task domain into this package.
type TaskOption struct {
	ID    string
	Title string
}

func (t TaskOption) Label() string {
	return fmt.Sprintf("[%s] %s", t.ID, t.Title)
}

// CloseTaskFunc closes a task with an optional outcome message. The
// tick service uses it for the post-session "mark as done?" prompt
// without depending on the task package directly.
type CloseTaskFunc func(id, outcome string) error

// LogTaskFunc appends a progress entry to a task. Called after a focus
// session targeted at a specific task ends, so the time spent is
// captured in the task's history regardless of the close choice.
type LogTaskFunc func(id, message string) error

func NewService(minutes int, tasks []TaskOption, closeTask CloseTaskFunc, logTask LogTaskFunc) *Service {
	return &Service{minutes: minutes, tasks: tasks, closeTask: closeTask, logTask: logTask}
}

type Service struct {
	minutes   int
	tasks     []TaskOption
	closeTask CloseTaskFunc
	logTask   LogTaskFunc
}

func (s *Service) Start() {
	r := newRootModel(s.minutes, s.tasks, s.closeTask, s.logTask)
	p := tea.NewProgram(r, tea.WithAltScreen())
	final, _ := p.Run()
	if fr, ok := final.(*rootModel); ok && fr.phase != phaseSetup {
		fmt.Println(fr.timer.Summary())
	}
}

const (
	phaseSetup = iota
	phaseTimer
	phaseReview
)

// rootModel hosts the setup form, timer, and post-session review prompt
// in a single Bubbletea program so the alt-screen is active from the
// very first frame.
type rootModel struct {
	minutes       int
	tasks         []TaskOption
	closeTask     CloseTaskFunc
	logTask       LogTaskFunc
	form          *huh.Form
	review        *huh.Form
	markDone      bool
	goal          string
	taskID        string
	taskLabel     string
	width, height int
	phase         int
	timer         model
}

func newRootModel(minutes int, tasks []TaskOption, closeTask CloseTaskFunc, logTask LogTaskFunc) *rootModel {
	r := &rootModel{minutes: minutes, tasks: tasks, closeTask: closeTask, logTask: logTask}
	fields := []huh.Field{
		huh.NewInput().
			Title("What's your goal for this session?").
			Placeholder("optional").
			CharLimit(50).
			Value(&r.goal),
	}
	if len(tasks) > 0 {
		opts := make([]huh.Option[string], 0, len(tasks)+1)
		opts = append(opts, huh.NewOption("(none)", ""))
		for _, t := range tasks {
			opts = append(opts, huh.NewOption(t.Label(), t.ID))
		}
		fields = append(fields, huh.NewSelect[string]().
			Title("Focus on a task").
			Options(opts...).
			Height(8).
			Value(&r.taskID))
	}
	r.form = huh.NewForm(huh.NewGroup(fields...)).
		WithShowHelp(true).
		WithWidth(setupFormWidth)
	return r
}

func (r *rootModel) Init() tea.Cmd { return r.form.Init() }

func (r *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		r.width, r.height = sz.Width, sz.Height
	}
	switch r.phase {
	case phaseSetup:
		return r.updateSetup(msg)
	case phaseTimer:
		if _, ok := msg.(timerExitMsg); ok {
			return r.finishTimer()
		}
		m2, cmd := r.timer.Update(msg)
		if mm, ok := m2.(model); ok {
			r.timer = mm
		}
		return r, cmd
	case phaseReview:
		return r.updateReview(msg)
	}
	return r, nil
}

func (r *rootModel) updateSetup(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := r.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		r.form = f
	}
	switch r.form.State {
	case huh.StateCompleted:
		return r, r.startTimer()
	case huh.StateAborted:
		return r, tea.Quit
	}
	return r, cmd
}

func (r *rootModel) updateReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := r.review.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		r.review = f
	}
	switch r.review.State {
	case huh.StateCompleted:
		if r.markDone && r.closeTask != nil {
			outcome := r.sessionLog()
			if outcome == "" {
				outcome = "Closed via tick focus session"
			} else {
				outcome = "Closed via tick — " + outcome
			}
			r.closeTask(r.taskID, outcome)
		} else {
			// kept open — record the session as a log entry instead
			r.logSession()
		}
		return r, tea.Quit
	case huh.StateAborted:
		r.logSession()
		return r, tea.Quit
	}
	return r, cmd
}

func (r *rootModel) startTimer() tea.Cmd {
	if r.taskID != "" {
		for _, t := range r.tasks {
			if t.ID == r.taskID {
				r.taskLabel = t.Label()
				break
			}
		}
	}
	m := newTimerModel(time.Duration(r.minutes)*time.Minute, r.goal, r.taskLabel)
	m.width, m.height = r.width, r.height
	r.timer = m
	r.phase = phaseTimer
	return r.timer.Init()
}

func (r *rootModel) finishTimer() (tea.Model, tea.Cmd) {
	if r.taskID == "" {
		return r, tea.Quit
	}
	if r.closeTask == nil {
		// no review possible — still capture the session as a log
		r.logSession()
		return r, tea.Quit
	}
	r.review = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Mark %s as done?", r.taskLabel)).
				Description("Goal: " + r.reviewGoal()).
				Affirmative("Yes, close it").
				Negative("No, keep open").
				Value(&r.markDone),
		),
	).WithShowHelp(true).WithWidth(setupFormWidth)
	r.phase = phaseReview
	return r, r.review.Init()
}

// logSession writes the session summary to the linked task, if any
// content and a logger are available. Called on review paths that do
// NOT close the task — closing already captures the same info via the
// outcome message.
func (r *rootModel) logSession() {
	if r.logTask == nil {
		return
	}
	if msg := r.sessionLog(); msg != "" {
		r.logTask(r.taskID, msg)
	}
}

// sessionLog returns a plain-text summary of the just-finished session
// suitable for storing in a task's log. Returns "" when no measurable
// time elapsed (user quit immediately) so we don't pollute the log.
func (r *rootModel) sessionLog() string {
	focused := r.timer.focused()
	if focused < time.Second {
		return ""
	}
	msg := fmt.Sprintf("Focus session: %s", formatDuration(focused))
	if r.goal != "" {
		msg += " · goal: " + r.goal
	}
	return msg
}

func (r *rootModel) reviewGoal() string {
	if r.goal == "" {
		return "(no goal set)"
	}
	return r.goal
}

func (r *rootModel) View() string {
	switch r.phase {
	case phaseTimer:
		return r.timer.View()
	case phaseReview:
		return r.centered(boxedForm("Session review", r.taskLabel, r.review.View()))
	}
	return r.centered(boxedForm("Focus session", fmt.Sprintf("%dm", r.minutes), r.form.View()))
}

func (r *rootModel) centered(s string) string {
	return lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center, s)
}

// boxedForm renders a huh form view inside the rounded setup box with a
// "title · meta" header above it.
func boxedForm(title, meta, formView string) string {
	header := setupTitle.Render(title) + " " + setupMeta.Render("· "+meta)
	body := lipgloss.JoinVertical(lipgloss.Left, header, "", strings.TrimRight(formView, "\n "))
	return setupBorder.Render(body)
}
