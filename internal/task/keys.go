package task

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	New    key.Binding
	Edit   key.Binding
	Delete key.Binding
	Up     key.Binding
	Down   key.Binding
	Right  key.Binding
	Left   key.Binding
	Enter  key.Binding
	Help   key.Binding
	Quit   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Enter},
		{k.New, k.Edit, k.Delete},
		{k.Help, k.Quit},
	}
}

type formKeyMap struct {
	Save key.Binding
	Help key.Binding
	Back key.Binding
	Quit key.Binding
}

func (k formKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Save, k.Back, k.Help, k.Quit}
}

func (k formKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Save, k.Back},
		{k.Help, k.Quit},
	}
}

var keys = keyMap{
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "navigate up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "navigate down"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "navigate right"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "navigate left"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "move task"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
}

// formKeys is used in forms - minimal keys for typing
var formKeys = formKeyMap{
	Save: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save"),
	),
	Help: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch field"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}
