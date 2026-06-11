package tuiproto

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/lucas-tremaroli/pace/internal/task"
)

const (
	minWidth  = 80
	minHeight = 20
)

// Root is the top-level Bubbletea model. It owns the tab strip,
// persistent header (storage + progress), tab body, and footer help.
type Root struct {
	tabs        []tab
	active      int
	width       int
	height      int
	storagePath string
	storageType storage.StorageType
	help        help.Model
	rootKeys    rootKeys
	tooSmall    bool
}

type rootKeys struct {
	Tab, ShiftTab, Quit key.Binding
}

func newRootKeys() rootKeys {
	return rootKeys{
		Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("⇥", "next tab")),
		ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("⇧⇥", "prev tab")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func New() (*Root, error) {
	ts, err := task.NewService()
	if err != nil {
		return nil, fmt.Errorf("task service: %w", err)
	}
	ns, err := note.NewService()
	if err != nil {
		ts.Close()
		return nil, fmt.Errorf("note service: %w", err)
	}

	var storePath string
	var storeType storage.StorageType
	if resolved, err := storage.ResolvePaceDir(); err == nil {
		storePath = resolved.Path
		storeType = resolved.Type
	}

	return &Root{
		tabs:        []tab{newKanban(ts), newNotes(ns)},
		help:        help.New(),
		rootKeys:    newRootKeys(),
		storagePath: storePath,
		storageType: storeType,
	}, nil
}

func (r *Root) Init() tea.Cmd {
	cmds := make([]tea.Cmd, len(r.tabs))
	for i, t := range r.tabs {
		cmds[i] = t.Init()
	}
	return tea.Batch(cmds...)
}

func (r *Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		r.width, r.height = m.Width, m.Height
		r.tooSmall = m.Width < minWidth || m.Height < minHeight
		r.help.Width = m.Width
		bodyMsg := tea.WindowSizeMsg{Width: m.Width, Height: r.bodyHeight()}
		cmds := make([]tea.Cmd, len(r.tabs))
		for i, t := range r.tabs {
			r.tabs[i], cmds[i] = t.Update(bodyMsg)
		}
		return r, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Filter or modal owns input first — root must not intercept
		// anything while the active tab is in a capturing state.
		if isCapturing(r.tabs[r.active]) {
			var cmd tea.Cmd
			r.tabs[r.active], cmd = r.tabs[r.active].Update(msg)
			return r, cmd
		}
		switch {
		case key.Matches(m, r.rootKeys.Quit):
			return r, tea.Quit
		case key.Matches(m, r.rootKeys.Tab):
			r.active = (r.active + 1) % len(r.tabs)
			return r, nil
		case key.Matches(m, r.rootKeys.ShiftTab):
			r.active = (r.active - 1 + len(r.tabs)) % len(r.tabs)
			return r, nil
		}
	}

	var cmd tea.Cmd
	r.tabs[r.active], cmd = r.tabs[r.active].Update(msg)
	return r, cmd
}

// isCapturing returns true when the active tab is in an input-capturing
// state (filter prompt open, modal open) and root must not intercept keys.
func isCapturing(t tab) bool {
	if t.ModalOverlay() != "" {
		return true
	}
	type filterer interface{ anyFiltering() bool }
	if f, ok := t.(filterer); ok {
		return f.anyFiltering()
	}
	type singleFilter interface{ inFilterMode() bool }
	if f, ok := t.(singleFilter); ok {
		return f.inFilterMode()
	}
	return false
}

func (n *notes) inFilterMode() bool {
	return n.list.FilterState() == list.Filtering
}

func (r *Root) View() string {
	if r.width == 0 {
		return "Loading..."
	}
	if r.tooSmall {
		msg := fmt.Sprintf("Terminal too small (%dx%d).\nMinimum: %dx%d.", r.width, r.height, minWidth, minHeight)
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 2).Render(msg)
	}
	header := r.renderHeader()
	body := r.tabs[r.active].View()
	footer := r.renderFooter()
	base := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	if overlay := r.tabs[r.active].ModalOverlay(); overlay != "" {
		return overlay
	}
	return base
}

func (r *Root) bodyHeight() int {
	h := r.height - r.headerHeight() - 1 // footer is 1 line
	if h < 1 {
		return 1
	}
	return h
}

func (r *Root) headerHeight() int { return 2 } // tab strip + spacer

func (r *Root) renderHeader() string {
	tabs := make([]string, len(r.tabs))
	for i, t := range r.tabs {
		if i == r.active {
			tabs[i] = theme.TabActive.Render(t.Title())
		} else {
			tabs[i] = theme.TabInactive.Render(t.Title())
		}
	}
	tabStrip := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	storeTag := theme.StorageTag.Render(strings.ToUpper(string(r.storageType)))
	storeLine := theme.OverviewLabel.Render(shortenPath(r.storagePath)) + "  " + storeTag

	// Right-align storage tag on the same row as the tab strip when there's room.
	gap := r.width - lipgloss.Width(tabStrip) - lipgloss.Width(storeLine)
	if gap < 1 {
		gap = 1
	}
	topRow := tabStrip + strings.Repeat(" ", gap) + storeLine

	return lipgloss.JoinVertical(lipgloss.Left, topRow, "")
}

func (r *Root) renderFooter() string {
	bindings := r.tabs[r.active].HelpBindings()
	bindings = append(bindings, r.rootKeys.Tab, r.rootKeys.Quit)
	km := footerKeys{bindings: bindings}
	return theme.Help.Render(r.help.View(km))
}

// footerKeys is a tiny help.KeyMap adapter so any slice of bindings can
// drive the footer.
type footerKeys struct{ bindings []key.Binding }

func (f footerKeys) ShortHelp() []key.Binding     { return f.bindings }
func (f footerKeys) FullHelp() [][]key.Binding    { return [][]key.Binding{f.bindings} }
