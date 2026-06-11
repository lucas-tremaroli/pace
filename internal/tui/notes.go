package tui

import (
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/note"
)

type notesService interface {
	ListNotesWithMeta(includeContent bool) ([]note.Note, error)
	ReadNoteWithMeta(filename string) (*note.Note, error)
	WriteNote(filename, content string) error
	GetNotePath(filename string) string
	DeleteNote(filename string) error
}

// --- messages ----------------------------------------------------------

type notesListedMsg struct {
	items []list.Item
	err   error
}

type notesMutatedMsg struct{}

type noteLoadedMsg struct {
	key     string
	content string
}

// --- model -------------------------------------------------------------

type notesKeys struct {
	Focus, Open, Filter, Reload, New, Edit, Delete key.Binding
}

func newNotesKeys() notesKeys {
	return notesKeys{
		Focus:  key.NewBinding(key.WithKeys("h", "left", "l", "right"), key.WithHelp("h/l", "focus")),
		Open:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", "open")),
		Filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Reload: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		New:    key.NewBinding(key.WithKeys("+"), key.WithHelp("+", "new note")),
		Edit:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete: key.NewBinding(key.WithKeys("backspace"), key.WithHelp("⌫", "delete")),
	}
}

func (k notesKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.New, k.Edit, k.Delete, k.Focus, k.Open, k.Filter, k.Reload}
}
func (k notesKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

const (
	notesFocusList   = 0
	notesFocusDetail = 1
)

type notes struct {
	svc           notesService
	list          list.Model
	vp            viewport.Model
	focus         int
	width         int
	height        int
	loadErr       error
	lastKey       string
	keys          notesKeys
	form          *NoteFormState
	confirm       *ConfirmFormState
	pendingSel    string
	pendingDelete string
}

func newNotes(svc notesService) *notes {
	l := list.New(nil, noteDelegate{}, 0, 0)
	l.Title = "Notes"
	l.Styles.Title = theme.ListTitle
	l.Styles.TitleBar = theme.ListTitleBar
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	vp := viewport.New(0, 0)
	vp.SetContent(theme.DetailDim.Render("Press ⏎ to load a note"))
	return &notes{svc: svc, list: l, vp: vp, keys: newNotesKeys()}
}

func (n *notes) Title() string         { return "Notes" }
func (n *notes) HelpBindings() []key.Binding { return n.keys.ShortHelp() }
func (n *notes) HelpHint() string {
	return "h/l focus • ⏎ open • / filter • j/k scroll"
}
func (n *notes) Count() int { return len(n.list.Items()) }

func (n *notes) Init() tea.Cmd {
	return func() tea.Msg {
		items, err := n.svc.ListNotesWithMeta(false)
		if err != nil {
			return notesListedMsg{err: err}
		}
		converted := make([]list.Item, len(items))
		for i, x := range items {
			converted[i] = noteItem{Note: x}
		}
		return notesListedMsg{items: converted}
	}
}

func (n *notes) Update(msg tea.Msg) (tab, tea.Cmd) {
	// Confirm dialog owns input while open.
	if n.confirm != nil {
		form, cmd := n.confirm.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			n.confirm.form = f
			switch n.confirm.form.State {
			case huh.StateCompleted:
				yes := n.confirm.result
				name := n.pendingDelete
				n.confirm = nil
				n.pendingDelete = ""
				if yes {
					return n, n.deleteCmd(name)
				}
				return n, nil
			case huh.StateAborted:
				n.confirm = nil
				n.pendingDelete = ""
				return n, nil
			}
		}
		return n, cmd
	}

	// Form owns input while open.
	if n.form != nil {
		form, cmd := n.form.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			n.form.form = f
			switch n.form.form.State {
			case huh.StateCompleted:
				openCmd := n.openEditorCmd()
				n.form = nil
				return n, openCmd
			case huh.StateAborted:
				n.form = nil
				return n, nil
			}
		}
		return n, cmd
	}

	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		n.width, n.height = m.Width, m.Height
		lw, dw, h := n.paneSizes()
		// inner = visual - border(2). Padding(0,1) is horizontal only.
		n.list.SetSize(lw-4, h-2)
		n.vp.Width = dw - 4
		n.vp.Height = h - 2
		return n, nil

	case notesListedMsg:
		n.loadErr = m.err
		n.list.SetItems(m.items)
		if n.pendingSel != "" {
			for i, it := range m.items {
				if ni, ok := it.(noteItem); ok && ni.Note.Filename == n.pendingSel {
					n.list.Select(i)
					break
				}
			}
			n.pendingSel = ""
		}
		return n, nil

	case editorFinishedMsg:
		// Force the next Enter on the same note to reload from disk;
		// otherwise loadCmd no-ops because filename == n.lastKey.
		n.lastKey = ""
		return n, tea.Batch(tea.ClearScreen, n.Init())

	case notesMutatedMsg:
		return n, n.Init()

	case noteLoadedMsg:
		n.lastKey = m.key
		n.vp.SetContent(m.content)
		n.vp.GotoTop()
		return n, nil

	case tea.KeyMsg:
		if n.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			n.list, cmd = n.list.Update(msg)
			return n, cmd
		}
		switch m.String() {
		case "h", "left":
			n.focus = notesFocusList
			return n, nil
		case "l", "right":
			n.focus = notesFocusDetail
			return n, nil
		case "enter":
			return n, n.loadCmd()
		case "r":
			return n, n.Init()
		case "e":
			return n, n.editCmd()
		case "backspace":
			it, ok := n.list.SelectedItem().(noteItem)
			if !ok {
				return n, nil
			}
			n.pendingDelete = it.Note.Filename
			n.confirm = newConfirmFormState(
				"Delete \""+it.Note.Filename+"\"?",
				"This will permanently delete the note.",
			)
			return n, n.confirm.form.Init()
		case "+":
			svc := n.svc
			n.form = newNoteFormState(func(name string) (bool, error) {
				path := svc.GetNotePath(name)
				_, err := os.Stat(path)
				if err == nil {
					return true, nil
				}
				if os.IsNotExist(err) {
					return false, nil
				}
				return false, err
			})
			return n, n.form.form.Init()
		}
		var cmd tea.Cmd
		if n.focus == notesFocusList {
			n.list, cmd = n.list.Update(msg)
		} else {
			n.vp, cmd = n.vp.Update(msg)
		}
		return n, cmd
	}

	// Forward unhandled messages to the list so filter matches,
	// blink, and pagination updates flow through.
	var lcmd, vcmd tea.Cmd
	n.list, lcmd = n.list.Update(msg)
	n.vp, vcmd = n.vp.Update(msg)
	return n, tea.Batch(lcmd, vcmd)
}

func (n *notes) loadCmd() tea.Cmd {
	it, ok := n.list.SelectedItem().(noteItem)
	if !ok {
		return nil
	}
	filename := it.Note.Filename
	if filename == n.lastKey {
		return nil
	}
	svc := n.svc
	w := n.detailContentWidth()
	return func() tea.Msg {
		full, err := svc.ReadNoteWithMeta(filename)
		if err != nil {
			return noteLoadedMsg{key: filename, content: theme.StatusBlocked.Render(err.Error())}
		}
		return noteLoadedMsg{key: filename, content: renderNoteDetail(*full, w)}
	}
}

// paneSizes returns the visual widths and height of the list and
// detail panes (border-inclusive). Widths sum to n.width and the
// height fills n.height so notes match the kanban box dimensions.
func (n *notes) paneSizes() (listW, detailW, h int) {
	listW = n.width / 3
	if listW < 22 {
		listW = 22
	}
	detailW = n.width - listW
	if detailW < 20 {
		detailW = 20
	}
	h = n.height
	if h < 6 {
		h = 6
	}
	return
}

func (n *notes) detailContentWidth() int {
	_, dw, _ := n.paneSizes()
	w := dw - 4 // border + padding
	if w < 20 {
		w = 20
	}
	return w
}

func (n *notes) View() string {
	if n.loadErr != nil {
		return theme.StatusBlocked.Render("error: " + n.loadErr.Error())
	}
	if n.width == 0 {
		return ""
	}
	lw, dw, h := n.paneSizes()

	listStyle := theme.BorderBlurred
	detailStyle := theme.BorderBlurred
	if n.focus == notesFocusList {
		listStyle = theme.BorderFocused
	} else {
		detailStyle = theme.BorderFocused
	}

	var listBody string
	if len(n.list.Items()) == 0 {
		listBody = theme.NoTasks.Render("  no notes")
	} else {
		listBody = n.list.View()
	}
	leftBox := listStyle.Width(lw - 2).Height(h - 2).Padding(0, 1).Render(listBody)
	rightBox := detailStyle.Width(dw - 2).Height(h - 2).Padding(0, 1).Render(n.vp.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}

func (n *notes) ModalOverlay() string {
	if n.confirm != nil {
		return boxedOverlay(n.confirm.form.View())
	}
	if n.form != nil {
		return boxedOverlay(n.form.form.View())
	}
	return ""
}

func (n *notes) deleteCmd(filename string) tea.Cmd {
	svc := n.svc
	return func() tea.Msg {
		svc.DeleteNote(filename)
		return notesMutatedMsg{}
	}
}

// editCmd opens the selected note in $EDITOR without writing a template.
func (n *notes) editCmd() tea.Cmd {
	it, ok := n.list.SelectedItem().(noteItem)
	if !ok {
		return nil
	}
	filename := strings.TrimSuffix(it.Note.Filename, ".md")
	n.pendingSel = it.Note.Filename
	path := n.svc.GetNotePath(filename)
	editor := resolveEditor()
	c := exec.Command(editor, path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})
}

// openEditorCmd writes a template for the new note and opens it in $EDITOR.
func (n *notes) openEditorCmd() tea.Cmd {
	filename := n.form.filename
	svc := n.svc
	editor := resolveEditor()
	path := svc.GetNotePath(filename)

	if strings.HasSuffix(filename, ".md") {
		n.pendingSel = filename
	} else {
		n.pendingSel = filename + ".md"
	}

	return func() tea.Msg {
		if err := svc.WriteNote(filename, note.DefaultTemplate(filename)); err != nil {
			return editorFinishedMsg{err}
		}
		c := exec.Command(editor, path)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return editorFinishedMsg{err}
		})()
	}
}
