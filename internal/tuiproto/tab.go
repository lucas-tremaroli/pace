package tuiproto

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// tab is the minimal contract a child model must satisfy to be hosted
// by Root. Each tab owns its own state, services, key handling, and
// rendering. The interface stays narrow on purpose; promotion to a
// definitive API can wait until a third tab arrives.
type tab interface {
	Init() tea.Cmd
	Update(tea.Msg) (tab, tea.Cmd)
	View() string

	Title() string
	HelpBindings() []key.Binding

	// ModalOverlay returns a full-screen string to layer on top of the
	// composed root view, or "" if nothing is open. Used for task
	// detail dialogs etc.
	ModalOverlay() string
}
