package tuiproto

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
}

// --- messages ----------------------------------------------------------

type kanbanLoadedMsg struct {
	items [3][]list.Item
	err   error
}

type taskMutatedMsg struct{}

type taskDetailLoadedMsg struct {
	title   string
	content string
}

// --- model -------------------------------------------------------------

type kanbanKeys struct {
	Left, Right, Open, Cycle, OpenLink, Filter, Reload, Quit key.Binding
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
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k kanbanKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Left, k.Right, k.Open, k.Cycle, k.OpenLink, k.Filter, k.Reload}
}
func (k kanbanKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

type kanban struct {
	svc     kanbanService
	cols    [3]list.Model
	col     int
	width   int
	height  int
	loadErr error
	keys    kanbanKeys
	modal   *modal
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
		k.modal = newModal(m.title, m.content, k.width, k.height)
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
	colW, colH := k.colInner()
	for i := range k.cols {
		k.cols[i].SetSize(colW, colH)
	}
}

func (k *kanban) colInner() (int, int) {
	colOuterW := k.width / 3
	// Account for rounded border (2) + padding (2).
	innerW := colOuterW - 4
	if innerW < 12 {
		innerW = 12
	}
	innerH := k.height - 4
	if innerH < 4 {
		innerH = 4
	}
	return innerW, innerH
}

func (k *kanban) View() string {
	if k.loadErr != nil {
		return theme.StatusBlocked.Render("error: " + k.loadErr.Error())
	}
	if k.width == 0 {
		return ""
	}
	var rendered [3]string
	colOuterW := k.width / 3
	for i := range k.cols {
		style := theme.BorderBlurred
		if i == k.col {
			style = theme.BorderFocused
		}
		body := k.cols[i].View()
		if len(k.cols[i].Items()) == 0 {
			body = theme.NoTasks.Render("  empty")
		}
		rendered[i] = style.
			Width(colOuterW - 2).
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
			title:   tk.ID() + " · " + tk.Title(),
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

// ModalOverlay returns the modal view if open, "" otherwise. Root uses
// this to render on top of the rest of the UI.
func (k *kanban) ModalOverlay() string {
	if k.modal == nil {
		return ""
	}
	return k.modal.View("")
}
