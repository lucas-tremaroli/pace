package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Tab      key.Binding
	Right    key.Binding
	Left     key.Binding
	Space    key.Binding
	Edit     key.Binding
	OpenLink key.Binding
	Delete   key.Binding
	New      key.Binding
	Quit     key.Binding
	Filter   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Left, k.Right, k.Filter, k.New, k.Delete, k.Edit, k.Space, k.OpenLink, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

var tuiKeys = keyMap{
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "tasks/notes"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "detail"),
	),
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "back"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "cycle status"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	OpenLink: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open link"),
	),
	Delete: key.NewBinding(
		key.WithKeys("backspace"),
		key.WithHelp("⌫", "delete"),
	),
	New: key.NewBinding(
		key.WithKeys("+"),
		key.WithHelp("+", "new"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
}
