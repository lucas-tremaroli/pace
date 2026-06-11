package tuiproto

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/lucas-tremaroli/pace/internal/task"
)

// kanbanService is the narrow interface the Kanban tab needs from the
// task service. Defined here so the tab owns its dependencies.
type kanbanService interface {
	LoadAllTasks() ([]task.Task, error)
	UpdateTask(t task.Task) error
	CloseTask(id, outcome string) error
	GetTaskLogs(id string) ([]storage.LogRecord, error)
	GenerateTaskID() string
	CreateTask(t task.Task) error
	SetLabel(id, label string) error
	DeleteTask(id string) error
	GetTaskByID(id string) (*task.Task, error)
}

// --- messages ----------------------------------------------------------

type kanbanLoadedMsg struct {
	items [3][]list.Item
	err   error
}

type taskMutatedMsg struct{}

type taskDetailLoadedMsg struct {
	content string
}

// --- model -------------------------------------------------------------

type kanbanKeys struct {
	Left, Right, Open, Cycle, OpenLink, Filter, Reload, New, Edit, Delete, Quit key.Binding
}

func newKanbanKeys() kanbanKeys {
	return kanbanKeys{
		Left:     key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "column")),
		Right:    key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "column")),
		Open:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", "details")),
		Cycle:    key.NewBinding(key.WithKeys(" "), key.WithHelp("␣", "cycle status")),
		OpenLink: key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open link")),
		Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Reload:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		New:      key.NewBinding(key.WithKeys("+"), key.WithHelp("+", "new task")),
		Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete:   key.NewBinding(key.WithKeys("backspace"), key.WithHelp("⌫", "delete")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k kanbanKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.New, k.Edit, k.Delete, k.Left, k.Right, k.Open, k.Cycle, k.OpenLink, k.Filter, k.Reload}
}
func (k kanbanKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

type kanban struct {
	svc           kanbanService
	cols          [3]list.Model
	col           int
	width         int
	height        int
	loadErr       error
	keys          kanbanKeys
	modal         *modal
	form          *TaskFormState
	confirm       *ConfirmFormState
	pendingDelete string
}

var columnTitles = [3]string{
	task.ColumnTitleTodo,
	task.ColumnTitleInProgress,
	task.ColumnTitleDone,
}

func newKanban(svc kanbanService) *kanban {
	var cols [3]list.Model
	for i := range cols {
		l := list.New(nil, taskDelegate{}, 0, 0)
		l.Title = columnTitles[i]
		l.Styles.Title = theme.ListTitle
		l.Styles.TitleBar = theme.ListTitleBar
		l.SetShowStatusBar(false)
		l.SetShowHelp(false)
		l.SetFilteringEnabled(true)
		l.DisableQuitKeybindings()
		cols[i] = l
	}
	return &kanban{svc: svc, cols: cols, keys: newKanbanKeys()}
}

func (k *kanban) Title() string         { return "Kanban" }
func (k *kanban) HelpBindings() []key.Binding { return k.keys.ShortHelp() }
func (k *kanban) Counts() (todo, inProg, done int) {
	return len(k.cols[0].Items()), len(k.cols[1].Items()), len(k.cols[2].Items())
}

func (k *kanban) HelpHint() string {
	return "h/l column • ⏎ details • ␣ cycle • / filter • o link"
}

func (k *kanban) Init() tea.Cmd { return k.load() }

func (k *kanban) load() tea.Cmd {
	return func() tea.Msg {
		tasks, err := k.svc.LoadAllTasks()
		if err != nil {
			return kanbanLoadedMsg{err: err}
		}
		sortTasks(tasks)
		var items [3][]list.Item
		for _, t := range tasks {
			s := int(t.Status())
			if s < 0 || s > 2 {
				continue
			}
			items[s] = append(items[s], taskItem{Task: t})
		}
		return kanbanLoadedMsg{items: items}
	}
}

func (k *kanban) anyFiltering() bool {
	for _, c := range k.cols {
		if c.FilterState() == list.Filtering {
			return true
		}
	}
	return false
}

func (k *kanban) selected() (task.Task, bool) {
	it, ok := k.cols[k.col].SelectedItem().(taskItem)
	if !ok {
		return task.Task{}, false
	}
	return it.Task, true
}

func (k *kanban) Update(msg tea.Msg) (tab, tea.Cmd) {
	// Confirm dialog owns input while open.
	if k.confirm != nil {
		form, cmd := k.confirm.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			k.confirm.form = f
			switch k.confirm.form.State {
			case huh.StateCompleted:
				yes := k.confirm.result
				id := k.pendingDelete
				k.confirm = nil
				k.pendingDelete = ""
				if yes {
					return k, k.deleteCmd(id)
				}
				return k, nil
			case huh.StateAborted:
				k.confirm = nil
				k.pendingDelete = ""
				return k, nil
			}
		}
		return k, cmd
	}

	// Form owns input while open.
	if k.form != nil {
		form, cmd := k.form.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			k.form.form = f
			switch k.form.form.State {
			case huh.StateCompleted:
				save := k.saveTaskCmd()
				k.form = nil
				return k, save
			case huh.StateAborted:
				k.form = nil
				return k, nil
			}
		}
		return k, cmd
	}

	// Modal owns input while open.
	if k.modal != nil {
		cmd := k.modal.Update(msg)
		if k.modal.Closed() {
			k.modal = nil
		}
		return k, cmd
	}

	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		k.width, k.height = m.Width, m.Height
		k.resizeColumns()
		return k, nil

	case kanbanLoadedMsg:
		k.loadErr = m.err
		for i := range k.cols {
			k.cols[i].SetItems(m.items[i])
			k.cols[i].Title = fmt.Sprintf("%s (%d)", columnTitles[i], len(m.items[i]))
		}
		return k, nil

	case taskMutatedMsg:
		return k, k.load()

	case taskDetailLoadedMsg:
		k.modal = newModal(m.content, k.width, k.height)
		return k, nil

	case tea.KeyMsg:
		if !k.anyFiltering() {
			switch {
			case key.Matches(m, k.keys.Left):
				if k.col > 0 {
					k.col--
				}
				return k, nil
			case key.Matches(m, k.keys.Right):
				if k.col < 2 {
					k.col++
				}
				return k, nil
			case key.Matches(m, k.keys.Reload):
				return k, k.load()
			case key.Matches(m, k.keys.Open):
				return k, k.openDetailCmd()
			case key.Matches(m, k.keys.Cycle):
				return k, k.cycleCmd()
			case key.Matches(m, k.keys.OpenLink):
				return k, k.openLinkCmd()
			case key.Matches(m, k.keys.New):
				k.form = newTaskFormState()
				return k, k.form.form.Init()
			case key.Matches(m, k.keys.Edit):
				tk, ok := k.selected()
				if !ok {
					return k, nil
				}
				k.form = newTaskEditFormState(tk)
				return k, k.form.form.Init()
			case key.Matches(m, k.keys.Delete):
				tk, ok := k.selected()
				if !ok {
					return k, nil
				}
				k.pendingDelete = tk.ID()
				k.confirm = newConfirmFormState(
					"Delete \""+tk.Title()+"\"?",
					"This will also remove its dependencies, links, and logs.",
				)
				return k, k.confirm.form.Init()
			}
		}
		var cmd tea.Cmd
		k.cols[k.col], cmd = k.cols[k.col].Update(msg)
		return k, cmd
	}

	// Forward unhandled messages (filter, spinner, blink, etc.) to all
	// columns so internal list state (filter matches, cursor blink)
	// continues to update.
	cmds := make([]tea.Cmd, 0, len(k.cols))
	for i := range k.cols {
		var cmd tea.Cmd
		k.cols[i], cmd = k.cols[i].Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return k, tea.Batch(cmds...)
}

func (k *kanban) resizeColumns() {
	widths := k.colWidths()
	innerH := k.colInnerH()
	for i := range k.cols {
		inner := widths[i] - 4 // border (2) + padding (2)
		if inner < 8 {
			inner = 8
		}
		k.cols[i].SetSize(inner, innerH)
	}
}

// colWidths returns three outer widths that sum to k.width. Remainder
// goes to the last column so totals are exact.
func (k *kanban) colWidths() [3]int {
	base := k.width / 3
	rem := k.width - base*3
	return [3]int{base, base, base + rem}
}

func (k *kanban) colInnerH() int {
	h := k.height - 4
	if h < 4 {
		return 4
	}
	return h
}

func (k *kanban) View() string {
	if k.loadErr != nil {
		return theme.StatusBlocked.Render("error: " + k.loadErr.Error())
	}
	if k.width == 0 {
		return ""
	}
	widths := k.colWidths()
	var rendered [3]string
	for i := range k.cols {
		style := theme.BorderBlurred
		if i == k.col {
			style = theme.BorderFocused
		}
		body := k.cols[i].View()
		if len(k.cols[i].Items()) == 0 {
			title := theme.ListTitle.Render(k.cols[i].Title)
			body = title + "\n\n" + theme.NoTasks.Render("  empty")
		}
		// Width(N) on a bordered style produces a box of total width N.
		rendered[i] = style.
			Width(widths[i] - 2).
			Height(k.height - 2).
			Padding(0, 1).
			Render(body)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered[0], rendered[1], rendered[2])
}

// --- commands ----------------------------------------------------------

func (k *kanban) openDetailCmd() tea.Cmd {
	tk, ok := k.selected()
	if !ok {
		return nil
	}
	svc := k.svc
	w := k.detailContentWidth()
	return func() tea.Msg {
		var logs []LogEntry
		if raw, err := svc.GetTaskLogs(tk.ID()); err == nil {
			logs = make([]LogEntry, len(raw))
			for i, l := range raw {
				logs[i] = LogEntry{Time: l.CreatedAt, IsOutcome: l.Type == "outcome", Message: l.Message}
			}
		}
		return taskDetailLoadedMsg{
			content: renderTaskDetail(tk, w, logs),
		}
	}
}

func (k *kanban) detailContentWidth() int {
	mw, _ := modalSize(k.width, k.height)
	w := mw - 8
	if w < 30 {
		w = 30
	}
	return w
}

func (k *kanban) cycleCmd() tea.Cmd {
	tk, ok := k.selected()
	if !ok || tk.IsBlocked() {
		return nil
	}
	next := tk.Status().GetNext()
	if err := tk.SetStatus(next); err != nil {
		return nil
	}
	svc := k.svc
	return func() tea.Msg {
		if next == task.Done {
			svc.CloseTask(tk.ID(), "")
		} else {
			svc.UpdateTask(tk)
		}
		return taskMutatedMsg{}
	}
}

func (k *kanban) openLinkCmd() tea.Cmd {
	tk, ok := k.selected()
	if !ok || tk.Link() == "" {
		return nil
	}
	link := tk.Link()
	return func() tea.Msg {
		var c *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			c = exec.Command("open", link)
		case "windows":
			c = exec.Command("rundll32", "url.dll,FileProtocolHandler", link)
		default:
			c = exec.Command("xdg-open", link)
		}
		c.Start()
		return nil
	}
}

// ModalOverlay returns the centered overlay (form or detail modal) if
// open, "" otherwise. Root uses this to render on top of the rest of
// the UI and to route input.
func (k *kanban) ModalOverlay() string {
	if k.confirm != nil {
		return boxedOverlay(k.confirm.form.View())
	}
	if k.form != nil {
		return boxedOverlay(k.form.form.View())
	}
	if k.modal == nil {
		return ""
	}
	return k.modal.View()
}

func (k *kanban) deleteCmd(id string) tea.Cmd {
	svc := k.svc
	return func() tea.Msg {
		svc.DeleteTask(id)
		return taskMutatedMsg{}
	}
}

func (k *kanban) saveTaskCmd() tea.Cmd {
	s := k.form
	svc := k.svc
	if s.id == "" {
		return func() tea.Msg {
			id := svc.GenerateTaskID()
			newTask := task.NewTaskComplete(id, task.Todo, s.title, s.desc, s.priority, s.link)
			newTask.SetLabel(s.label)
			if err := svc.CreateTask(newTask); err != nil {
				return taskMutatedMsg{}
			}
			svc.SetLabel(id, s.label)
			return taskMutatedMsg{}
		}
	}
	return func() tea.Msg {
		existing, err := svc.GetTaskByID(s.id)
		if err != nil {
			return taskMutatedMsg{}
		}
		updated := task.NewTaskComplete(existing.ID(), existing.Status(), s.title, s.desc, s.priority, s.link)
		updated.SetLabel(s.label)
		updated.SetBlockedBy(existing.BlockedBy())
		updated.SetBlocks(existing.Blocks())
		updated.SetNotes(existing.Notes())
		svc.UpdateTask(updated)
		svc.SetLabel(s.id, s.label)
		return taskMutatedMsg{}
	}
}
