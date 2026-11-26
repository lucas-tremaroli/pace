package cmdutil

import (
	"os"
	"os/exec"
)

// ResolveEditor picks an editor, preferring $EDITOR, then nvim/vim/vi/nano.
func ResolveEditor() string {
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

// EditText opens content in the user's editor via a temp .md file and returns
// the edited text.
func EditText(content string) (string, error) {
	f, err := os.CreateTemp("", "pace-epic-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command(ResolveEditor(), f.Name())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	out, err := os.ReadFile(f.Name())
	return string(out), err
}

// KeepText saves text to a temp file and returns its path, so an edit that
// failed validation isn't lost. Returns "" if the file can't be written.
func KeepText(text string) string {
	f, err := os.CreateTemp("", "pace-epic-rejected-*.md")
	if err != nil {
		return ""
	}
	defer f.Close()
	if _, err := f.WriteString(text); err != nil {
		return ""
	}
	return f.Name()
}
