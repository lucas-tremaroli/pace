package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/task"
)

type epicsService interface {
	LoadAllEpics() ([]epic.Epic, error)
	GenerateEpicID() string
	CreateEpic(e epic.Epic) error
	DeleteEpic(id string) error
}

// epicTasksService is the slice of the task service the epics tab needs to
// show an epic's tasks.
type epicTasksService interface {
	LoadAllTasks() ([]task.Task, error)
}

// --- messages ----------------------------------------------------------

type epicsListedMsg struct {
	items []list.Item
	err   error
}

type epicsMutatedMsg struct{}

type epicLoadedMsg struct {
	key     string
	content string
}

// --- model -------------------------------------------------------------

type epicsKeys struct {
	Focus, Open, Filter, Reload, New, Delete key.Binding
}

func newEpicsKeys() epicsKeys {
	return epicsKeys{
		Focus:  key.NewBinding(key.WithKeys("h", "left", "l", "right"), key.WithHelp("h/l", "focus")),
		Open:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", "open")),
		Filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Reload: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		New:    key.NewBinding(key.WithKeys("+"), key.WithHelp("+", "new epic")),
		Delete: key.NewBinding(key.WithKeys("backspace"), key.WithHelp("⌫", "delete")),
	}
}

func (k epicsKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.New, k.Delete, k.Focus, k.Open, k.Filter, k.Reload}
}
func (k epicsKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

const (
	epicsFocusList   = 0
	epicsFocusDetail = 1
)

type epics struct {
	svc           epicsService
	tasks         epicTasksService
	list          list.Model
	vp            viewport.Model
	spinner       spinner.Model
	loading       bool
	loadingKey    string
	focus         int
	width         int
	height        int
	loadErr       error
	lastKey       string
	keys          epicsKeys
	form          *EpicFormState
	confirm       *ConfirmFormState
	pendingDelete string
}

func newEpics(svc epicsService, tasks epicTasksService) *epics {
	l := list.New(nil, epicDelegate{}, 0, 0)
	l.Title = "Epics"
	l.Styles.Title = theme.ListTitle
	l.Styles.TitleBar = theme.ListTitleBar
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	vp := viewport.New(0, 0)
	vp.SetContent(theme.DetailDim.Render("Press ⏎ to load an epic"))
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = theme.DetailDim
	return &epics{svc: svc, tasks: tasks, list: l, vp: vp, spinner: sp, keys: newEpicsKeys()}
}

func (e *epics) Title() string               { return "Epics" }
func (e *epics) HelpBindings() []key.Binding { return e.keys.ShortHelp() }
func (e *epics) HelpHint() string {
	return "h/l focus • ⏎ open • / filter • j/k scroll"
}
func (e *epics) Count() int { return len(e.list.Items()) }

func (e *epics) Init() tea.Cmd {
	svc := e.svc
	return func() tea.Msg {
		all, err := svc.LoadAllEpics()
		if err != nil {
			return epicsListedMsg{err: err}
		}
		items := make([]list.Item, len(all))
		for i, x := range all {
			items[i] = epicItem{Epic: x}
		}
		return epicsListedMsg{items: items}
	}
}

func (e *epics) Update(msg tea.Msg) (tab, tea.Cmd) {
	// Confirm dialog owns input while open.
	if e.confirm != nil {
		form, cmd := e.confirm.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			e.confirm.form = f
			switch e.confirm.form.State {
			case huh.StateCompleted:
				yes := e.confirm.result
				id := e.pendingDelete
				e.confirm = nil
				e.pendingDelete = ""
				if yes {
					return e, e.deleteCmd(id)
				}
				return e, nil
			case huh.StateAborted:
				e.confirm = nil
				e.pendingDelete = ""
				return e, nil
			}
		}
		return e, cmd
	}

	// Form owns input while open.
	if e.form != nil {
		form, cmd := e.form.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			e.form.form = f
			switch e.form.form.State {
			case huh.StateCompleted:
				create := e.createCmd(e.form.title)
				e.form = nil
				return e, create
			case huh.StateAborted:
				e.form = nil
				return e, nil
			}
		}
		return e, cmd
	}

	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		e.width, e.height = m.Width, m.Height
		lw, dw, h := e.paneSizes()
		// inner = visual - border(2). Padding(0,1) is horizontal only.
		e.list.SetSize(lw-4, h-2)
		e.vp.Width = dw - 4
		e.vp.Height = h - 2
		return e, nil

	case epicsListedMsg:
		e.loadErr = m.err
		e.list.SetItems(m.items)
		return e, nil

	case epicsMutatedMsg:
		// Force the next Enter to reload; the epic may be gone or changed.
		e.lastKey = ""
		return e, e.Init()

	case epicLoadedMsg:
		e.loading = false
		e.loadingKey = ""
		e.lastKey = m.key
		e.vp.SetContent(m.content)
		e.vp.GotoTop()
		return e, nil

	case spinner.TickMsg:
		if !e.loading {
			return e, nil
		}
		var cmd tea.Cmd
		e.spinner, cmd = e.spinner.Update(msg)
		return e, cmd

	case tea.KeyMsg:
		if e.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			e.list, cmd = e.list.Update(msg)
			return e, cmd
		}
		switch m.String() {
		case "h", "left":
			e.focus = epicsFocusList
			return e, nil
		case "l", "right":
			e.focus = epicsFocusDetail
			return e, nil
		case "enter":
			return e, e.loadCmd()
		case "r":
			e.lastKey = ""
			return e, e.Init()
		case "backspace":
			it, ok := e.list.SelectedItem().(epicItem)
			if !ok {
				return e, nil
			}
			e.pendingDelete = it.Epic.ID()
			e.confirm = newConfirmFormState(
				"Delete \""+it.Epic.Title()+"\"?",
				"Its tasks are kept but unlinked from the epic.",
			)
			return e, e.confirm.form.Init()
		case "+":
			e.form = newEpicFormState()
			return e, e.form.form.Init()
		}
		var cmd tea.Cmd
		if e.focus == epicsFocusList {
			e.list, cmd = e.list.Update(msg)
		} else {
			e.vp, cmd = e.vp.Update(msg)
		}
		return e, cmd
	}

	// Forward unhandled messages to the list so filter matches,
	// blink, and pagination updates flow through.
	var lcmd, vcmd tea.Cmd
	e.list, lcmd = e.list.Update(msg)
	e.vp, vcmd = e.vp.Update(msg)
	return e, tea.Batch(lcmd, vcmd)
}

// loadCmd fetches the selected epic's tasks and renders the detail pane.
// Tasks are only read here (on Enter), never on navigation.
func (e *epics) loadCmd() tea.Cmd {
	it, ok := e.list.SelectedItem().(epicItem)
	if !ok {
		return nil
	}
	id := it.Epic.ID()
	if id == e.lastKey || (e.loading && id == e.loadingKey) {
		return nil
	}
	selected := it.Epic
	tasks := e.tasks
	w := e.detailContentWidth()
	e.loading = true
	e.loadingKey = id
	read := func() tea.Msg {
		all, err := tasks.LoadAllTasks()
		if err != nil {
			return epicLoadedMsg{key: id, content: theme.StatusBlocked.Render(err.Error())}
		}
		var members []task.Task
		for _, t := range all {
			if t.EpicID() == id {
				members = append(members, t)
			}
		}
		return epicLoadedMsg{key: id, content: renderEpicDetail(selected, members, w)}
	}
	return tea.Batch(e.spinner.Tick, read)
}

// paneSizes returns the visual widths and height of the list and
// detail panes (border-inclusive). Widths sum to e.width and the
// height fills e.height so epics match the kanban box dimensions.
func (e *epics) paneSizes() (listW, detailW, h int) {
	listW = e.width / 3
	if listW < 22 {
		listW = 22
	}
	detailW = e.width - listW
	if detailW < 20 {
		detailW = 20
	}
	h = e.height
	if h < 6 {
		h = 6
	}
	return
}

func (e *epics) detailContentWidth() int {
	_, dw, _ := e.paneSizes()
	w := dw - 4 // border + padding
	if w < 20 {
		w = 20
	}
	return w
}

func (e *epics) inFilterMode() bool {
	return e.list.FilterState() == list.Filtering
}

func (e *epics) View() string {
	if e.loadErr != nil {
		return theme.StatusBlocked.Render("error: " + e.loadErr.Error())
	}
	if e.width == 0 {
		return ""
	}
	lw, dw, h := e.paneSizes()

	listStyle := theme.BorderBlurred
	detailStyle := theme.BorderBlurred
	if e.focus == epicsFocusList {
		listStyle = theme.BorderFocused
	} else {
		detailStyle = theme.BorderFocused
	}

	var listBody string
	if len(e.list.Items()) == 0 {
		listBody = theme.NoTasks.Render("  no epics")
	} else {
		listBody = e.list.View()
	}
	leftBox := listStyle.Width(lw-2).Height(h-2).Padding(0, 1).Render(listBody)
	var detailBody string
	if e.loading {
		detailBody = e.spinner.View() + theme.DetailDim.Render(" Loading "+e.loadingKey+"…")
	} else {
		detailBody = e.vp.View()
	}
	rightBox := detailStyle.Width(dw-2).Height(h-2).Padding(0, 1).Render(detailBody)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}

func (e *epics) ModalOverlay() string {
	if e.confirm != nil {
		return boxedOverlay(e.confirm.form.View())
	}
	if e.form != nil {
		return boxedOverlay(e.form.form.View())
	}
	return ""
}

func (e *epics) deleteCmd(id string) tea.Cmd {
	svc := e.svc
	return func() tea.Msg {
		svc.DeleteEpic(id)
		return epicsMutatedMsg{}
	}
}

func (e *epics) createCmd(title string) tea.Cmd {
	svc := e.svc
	return func() tea.Msg {
		if err := svc.CreateEpic(epic.NewEpic(svc.GenerateEpicID(), epic.Planning, title, "")); err != nil {
			return epicLoadedMsg{content: theme.StatusBlocked.Render(fmt.Sprintf("create epic: %v", err))}
		}
		return epicsMutatedMsg{}
	}
}
