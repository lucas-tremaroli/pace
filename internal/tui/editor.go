package tui

import (
	"os"
	"os/exec"
)

// editorFinishedMsg is sent when the external editor process exits.
type editorFinishedMsg struct{ err error }

// resolveEditor picks an editor, preferring $EDITOR, then nvim/vim/vi/nano.
func resolveEditor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		if _, err := exec.LookPath(e); err == nil {
			return e
		}
	}
	for _, e := range []string{"nvim", "vim", "vi", "nano"} {
		if _, err := exec.LookPath(e); err == nil {
			return e
		}
	}
	return "vi"
}
