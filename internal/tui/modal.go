package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// modal is a centered overlay that wraps a viewport for scrollable
// content. It owns input while open and exits on esc/q.
type modal struct {
	vp     viewport.Model
	width  int // outer screen width
	height int // outer screen height
	closed bool
}

func newModal(content string, screenW, screenH int) *modal {
	mw, mh := modalSize(screenW, screenH)
	vp := viewport.New(mw-4, mh-4)
	vp.SetContent(content)
	return &modal{vp: vp, width: screenW, height: screenH}
}

func modalSize(w, h int) (int, int) {
	mw := w * 4 / 5
	if mw > 100 {
		mw = 100
	}
	if mw < 40 {
		mw = 40
	}
	mh := h * 4 / 5
	if mh < 12 {
		mh = 12
	}
	return mw, mh
}

func (m *modal) Closed() bool { return m.closed }

func (m *modal) Update(msg tea.Msg) tea.Cmd {
	switch t := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = t.Width, t.Height
		mw, mh := modalSize(t.Width, t.Height)
		m.vp.Width = mw - 4
		m.vp.Height = mh - 4
		return nil
	case tea.KeyMsg:
		switch t.String() {
		case "esc", "q":
			m.closed = true
			return nil
		case "j", "down":
			m.vp.ScrollDown(1)
		case "k", "up":
			m.vp.ScrollUp(1)
		case "g", "home":
			m.vp.GotoTop()
		case "G", "end":
			m.vp.GotoBottom()
		case "u", "ctrl+u":
			m.vp.HalfPageUp()
		case "d", "ctrl+d":
			m.vp.HalfPageDown()
		}
	}
	return nil
}

var modalBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("62")).
	Padding(1, 2)

// View returns the raw modal box; root.View does the screen placement.
func (m *modal) View() string {
	mw, mh := modalSize(m.width, m.height)
	return modalBorder.Width(mw - 4).Height(mh - 2).Render(m.vp.View())
}

// boxedOverlay wraps a huh form view in the same rounded border used
// by the task detail modal. Trailing whitespace from the form is
// trimmed so lipgloss can compute the box height (and root can center
// it) accurately. Visual size stays consistent regardless of the
// message length because the form drives its own width via WithWidth.
func boxedOverlay(content string) string {
	trimmed := strings.TrimRight(content, "\n ")
	return modalBorder.Render(trimmed)
}
