package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Tab         key.Binding
	Enter       key.Binding
	Left        key.Binding
	Right       key.Binding
	Delete      key.Binding
	Edit        key.Binding
	StatusNext  key.Binding
	StatusPrev  key.Binding
	StatusCycle key.Binding // display-only combined help entry
	Quit        key.Binding
	Filter      key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Enter, k.Left, k.Right, k.Delete, k.Edit, k.StatusCycle, k.Filter, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

var tuiKeys = keyMap{
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "tasks/notes"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open detail"),
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
	Edit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "edit (detail)"),
	),
	StatusNext: key.NewBinding(
		key.WithKeys("shift+right"),
	),
	StatusPrev: key.NewBinding(
		key.WithKeys("shift+left"),
	),
	StatusCycle: key.NewBinding(
		key.WithKeys("shift+right", "shift+left"),
		key.WithHelp("shift+←/→", "cycle status"),
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
