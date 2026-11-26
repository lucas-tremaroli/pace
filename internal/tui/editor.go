package tui

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/epic"
)

// epicEditedMsg is sent when the external editor exits; path holds the edited
// document and base the epic it was opened from.
type epicEditedMsg struct {
	path   string
	base   epic.Epic
	create bool
	err    error
}

// editEpicCmd writes base to a temp file, opens it in $EDITOR and reports back
// with epicEditedMsg. The file is read and removed by saveEditedEpicCmd.
func editEpicCmd(base epic.Epic, create bool) tea.Cmd {
	f, err := os.CreateTemp("", "pace-epic-*.md")
	if err != nil {
		return func() tea.Msg { return epicEditedMsg{err: err} }
	}
	_, werr := f.WriteString(epic.Format(base))
	f.Close()
	if werr != nil {
		os.Remove(f.Name())
		return func() tea.Msg { return epicEditedMsg{err: werr} }
	}
	path := f.Name()
	c := exec.Command(cmdutil.ResolveEditor(), path)
	return tea.Sequence(tea.ClearScreen, tea.ExecProcess(c, func(err error) tea.Msg {
		return epicEditedMsg{path: path, base: base, create: create, err: err}
	}))
}
