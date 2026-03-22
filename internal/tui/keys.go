package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Tab    key.Binding
	Left   key.Binding
	Right  key.Binding
	Delete key.Binding
	Quit   key.Binding
	Filter key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Left, k.Right, k.Delete, k.Filter, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

var tuiKeys = keyMap{
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "tasks/notes"),
	),
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "lists"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "detail"),
	),
	Delete: key.NewBinding(
		key.WithKeys("backspace"),
		key.WithHelp("⌫", "delete"),
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
