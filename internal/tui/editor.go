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
// with epicEditedMsg. The temp file is created inside the Cmd (no I/O in
// Update) and is read and removed by saveEditedCmd.
func editEpicCmd(base epic.Epic, create bool) tea.Cmd {
	open := func() tea.Msg {
		f, err := os.CreateTemp("", "pace-epic-*.md")
		if err != nil {
			return epicEditedMsg{err: err}
		}
		_, werr := f.WriteString(epic.Format(base))
		f.Close()
		if werr != nil {
			os.Remove(f.Name())
			return epicEditedMsg{err: werr}
		}
		path := f.Name()
		c := exec.Command(cmdutil.ResolveEditor(), path)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return epicEditedMsg{path: path, base: base, create: create, err: err}
		})()
	}
	return tea.Sequence(tea.ClearScreen, open)
}
