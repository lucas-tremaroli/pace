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
	vp := viewport.New(modalInner(screenW, screenH))
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

// modalOuter is the modal box's outer size on a screen of the given size.
func modalOuter(screenW, screenH int) (int, int) {
	mw, mh := modalSize(screenW, screenH)
	return mw - 2, mh
}

// modalInner is the scrollable content area inside the modal's frame.
func modalInner(screenW, screenH int) (int, int) {
	ow, oh := modalOuter(screenW, screenH)
	return paneInner(modalBorder, ow, oh)
}

func (m *modal) Closed() bool { return m.closed }

func (m *modal) Update(msg tea.Msg) tea.Cmd {
	switch t := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = t.Width, t.Height
		m.vp.Width, m.vp.Height = modalInner(t.Width, t.Height)
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
	ow, oh := modalOuter(m.width, m.height)
	return pane(modalBorder, ow, oh, m.vp.View())
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
