package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"

	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/lucas-tremaroli/pace/internal/task"
)

const (
	focusTasks        = 0
	focusNotes        = 1
	focusDetail       = 2
	minListW          = 30
	minWidth          = 80
	minHeight         = 20
	dialogWidth       = 50
	overviewH         = 7 // pad + storage + pad + bar + pad + counts + pad
	detailPlaceholder = "Press → to view details"
)

var (
	focusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
	blurredBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("236"))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)
	listTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))
	listTitleBarStyle = lipgloss.NewStyle().MarginBottom(1)

	// Detail panel styles
	detailHeader   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)
	detailTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	detailID       = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	detailDesc     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	detailLabel    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	detailValue    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	detailDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	detailLogTime  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	detailOutcome  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	// Section headers
	detailSection = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)

	// Field box styles
	fieldBoxBorder = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	fieldBoxLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	chipBracket    = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	chipText       = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	chipLabel      = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	detailLinkLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	detailLinkURL   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	statusTodo     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	statusProgress = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	statusDone     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusBlocked  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	priorityP1     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	priorityP2     = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	priorityP3     = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))

	// Overview styles
	overviewLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	overviewPct    = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	storageTag     = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	progressFilled = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	progressEmpty  = lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	noTasksStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
)


type Tui struct {
	taskList    list.Model
	noteList    list.Model
	viewport    viewport.Model
	help        help.Model
	taskService *task.Service
	noteService *note.Service
	storagePath string
	storageType storage.StorageType
	focus       int
	width       int
	height      int
	loaded      bool
	quitting    bool
	tooSmall    bool
	lastKey        string
	lastListFocus  int // tracks which list was focused before entering detail
	detailSeq      uint64
	spinner        spinner.Model
	detailLoading  bool
	confirmForm   *huh.Form
	confirmResult *bool
	deleteTarget  string // "task" or "note"

	taskForm     *huh.Form
	formTaskID   string // non-empty for edit, empty for create
	formTitle    string
	formDesc     string
	formLink     string
	formLabel    string
	formPriority int

	noteForm          *huh.Form
	formNoteFilename  string
	pendingNoteSelect string // filename to select after data reload (note create)

	// Cached layout dimensions, updated by recalcLayout.
	layoutAvailH    int
	layoutTaskH     int
	layoutNoteH     int
	cachedHelpH     int
	cachedHelpWidth int

}

func newList(title string, items []list.Item, delegate list.ItemDelegate) list.Model {
	l := list.New(items, delegate, 0, 0)
	l.Title = title
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = listTitleBarStyle
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	return l
}

func NewTui() (*Tui, error) {
	taskService, err := task.NewService()
	if err != nil {
		return nil, fmt.Errorf("failed to init task service: %w", err)
	}

	noteService, err := note.NewService()
	if err != nil {
		taskService.Close()
		return nil, fmt.Errorf("failed to init note service: %w", err)
	}

	vp := viewport.New(0, 0)
	vp.SetContent(detailPlaceholder)

	var storePath string
	var storeType storage.StorageType
	if resolved, err := storage.ResolvePaceDir(); err == nil {
		storePath = resolved.Path
		storeType = resolved.Type
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	noteList := newList("Notes", nil, noteDelegate{})
	noteList.SetFilteringEnabled(false)

	return &Tui{
		taskList:    newList("Tasks", nil, taskDelegate{}),
		noteList:    noteList,
		viewport:    vp,
		help:        help.New(),
		taskService: taskService,
		noteService: noteService,
		spinner:     sp,
		storagePath: storePath,
		storageType: storeType,
		focus:       focusTasks,
	}, nil
}

func (t *Tui) Init() tea.Cmd {
	return t.loadDataCmd()
}

func (t *Tui) isFiltering() bool {
	return t.taskList.FilterState() == list.Filtering ||
		t.noteList.FilterState() == list.Filtering
}

// syncFilterEnabled enables filtering only on the focused list and disables
// it on the other, preventing both lists from entering filter mode and
// conflicting with each other.
func (t *Tui) syncFilterEnabled() {
	t.taskList.SetFilteringEnabled(t.focus == focusTasks)
	t.noteList.SetFilteringEnabled(t.focus == focusNotes)
}

func (t *Tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always track window size, even when a form overlay is active.
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		t.width = ws.Width
		t.height = ws.Height
		t.tooSmall = ws.Width < minWidth || ws.Height < minHeight
		if !t.tooSmall {
			t.recalcLayout()
		}
		t.loaded = true
	}

	// Handle confirm dialog when active
	if t.confirmForm != nil {
		form, cmd := t.confirmForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			t.confirmForm = f
			if t.confirmForm.State == huh.StateCompleted {
				t.confirmForm = nil
				confirmed := t.confirmResult != nil && *t.confirmResult
				t.confirmResult = nil
				target := t.deleteTarget
				t.deleteTarget = ""
				if confirmed {
					return t, t.deleteCmd(target)
				}
				return t, nil
			}
			if t.confirmForm.State == huh.StateAborted {
				t.confirmForm = nil
				t.confirmResult = nil
				t.deleteTarget = ""
				return t, nil
			}
		}
		return t, cmd
	}

	// Handle task form (create/edit) when active
	if t.taskForm != nil {
		form, cmd := t.taskForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			t.taskForm = f
			if t.taskForm.State == huh.StateCompleted {
				saveCmd := t.taskFormSaveCmd()
				t.taskForm = nil
				t.formTaskID = ""
				return t, saveCmd
			}
			if t.taskForm.State == huh.StateAborted {
				t.taskForm = nil
				t.formTaskID = ""
				return t, nil
			}
		}
		return t, cmd
	}

	// Handle note form (create) when active
	if t.noteForm != nil {
		form, cmd := t.noteForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			t.noteForm = f
			if t.noteForm.State == huh.StateCompleted {
				openCmd := t.noteFormOpenEditorCmd()
				t.noteForm = nil
				return t, openCmd
			}
			if t.noteForm.State == huh.StateAborted {
				t.noteForm = nil
				t.formNoteFilename = ""
				return t, nil
			}
		}
		return t, cmd
	}

	switch msg := msg.(type) {
	case editorFinishedMsg:
		// Reload data to pick up any changes made in the editor.
		// On error (e.g. editor not found), we still reload since the note
		// file may have been created before the editor failed.
		t.lastKey = "" // force detail re-render
		return t, t.loadDataCmd()

	case dataReloadedMsg:
		noteIdx := t.noteList.Index()
		t.taskList.SetItems(msg.tasks)
		t.noteList.SetItems(msg.notes)
		t.taskList.Title = fmt.Sprintf("Tasks (%d)", len(msg.tasks))
		t.noteList.Title = fmt.Sprintf("Notes (%d)", len(msg.notes))
		if t.pendingNoteSelect != "" {
			for i, item := range msg.notes {
				if ni, ok := item.(NoteItem); ok && ni.Note.Filename == t.pendingNoteSelect {
					t.noteList.Select(i)
					noteIdx = i
					break
				}
			}
			t.pendingNoteSelect = ""
		} else if noteIdx >= len(msg.notes) && len(msg.notes) > 0 {
			t.noteList.Select(len(msg.notes) - 1)
		}
		// Re-sync list sizes so pagination is calculated with the new items
		if t.loaded && !t.tooSmall {
			t.recalcLayout()
		}
		hadDetail := t.lastKey != "" || t.focus == focusDetail
		t.lastKey = "" // force re-render on next open or below
		if hadDetail {
			return t, t.refreshDetailCmd()
		}
		return t, nil

	case taskStatusUpdatedMsg:
		// Find the task by ID to update in-place, since the index captured at
		// cycle time can become stale (filtering, reloads, tab switching).
		for i, item := range t.taskList.Items() {
			if ti, ok := item.(TaskItem); ok && ti.Task.ID() == msg.task.ID() {
				t.taskList.SetItem(i, TaskItem{Task: msg.task})
				break
			}
		}
		hadDetail := t.lastKey != ""
		t.lastKey = "" // force re-render with updated data
		if hadDetail {
			return t, t.refreshDetailCmd()
		}
		return t, nil

	case detailRenderedMsg:
		if msg.seq != t.detailSeq {
			return t, nil // stale render from an older request; discard
		}
		t.detailLoading = false
		t.lastKey = msg.key
		t.viewport.SetContent(msg.content)
		t.viewport.GotoTop()
		return t, nil

	case spinner.TickMsg:
		if t.detailLoading {
			var cmd tea.Cmd
			t.spinner, cmd = t.spinner.Update(msg)
			return t, cmd
		}
		return t, nil

	case tea.WindowSizeMsg:
		// Size tracking already handled above; just refresh detail if needed.
		t.lastKey = "" // force detail re-render at new width
		if !t.tooSmall && t.focus == focusDetail {
			return t, t.refreshDetailCmd()
		}
		return t, nil

	case tea.KeyMsg:
		// When a list is filtering, all keys go to it
		if t.isFiltering() {
			return t.updateFilteringList(msg)
		}

		switch {
		case key.Matches(msg, tuiKeys.Quit):
			t.quitting = true
			if t.taskService != nil {
				t.taskService.Close()
			}
			if t.noteService != nil {
				t.noteService.Close()
			}
			return t, tea.Quit

		case key.Matches(msg, tuiKeys.New):
			if t.focus == focusTasks {
				return t.startCreate()
			}
			if t.focus == focusNotes {
				return t.startNoteCreate()
			}

		case key.Matches(msg, tuiKeys.Delete):
			return t.startDelete()

		case key.Matches(msg, tuiKeys.Tab):
			if t.focus == focusDetail {
				return t, nil
			}
			if t.focus == focusTasks {
				t.focus = focusNotes
			} else {
				t.focus = focusTasks
			}
			t.syncFilterEnabled()
			return t, nil

		case key.Matches(msg, tuiKeys.Right):
			if t.focus != focusDetail {
				t.lastListFocus = t.focus
				cmd := t.refreshDetailCmd()
				t.focus = focusDetail
				return t, cmd
			}

		case key.Matches(msg, tuiKeys.Left):
			if t.focus == focusDetail {
				t.focus = t.lastListFocus
				t.syncFilterEnabled()
				return t, nil
			}

		case key.Matches(msg, tuiKeys.Edit):
			return t.startEdit()

		case key.Matches(msg, tuiKeys.OpenLink):
			return t, t.openLinkCmd()

		case key.Matches(msg, tuiKeys.Space):
			if t.focus == focusTasks || (t.focus == focusDetail && t.lastListFocus == focusTasks) {
				return t, t.cycleTaskStatusCmd(true)
			}
		}

		switch t.focus {
		case focusTasks:
			var cmd tea.Cmd
			t.taskList, cmd = t.taskList.Update(msg)
			return t, cmd
		case focusNotes:
			var cmd tea.Cmd
			t.noteList, cmd = t.noteList.Update(msg)
			return t, cmd
		case focusDetail:
			return t.updateViewport(msg)
		}
	}

	// Forward non-key messages (blink, etc.) to the focused list only
	var cmd tea.Cmd
	switch t.focus {
	case focusTasks:
		t.taskList, cmd = t.taskList.Update(msg)
	case focusNotes:
		t.noteList, cmd = t.noteList.Update(msg)
	}
	return t, cmd
}

func (t *Tui) startDelete() (tea.Model, tea.Cmd) {
	var title, description string

	switch t.focus {
	case focusTasks:
		item, ok := t.taskList.SelectedItem().(TaskItem)
		if !ok {
			return t, nil
		}
		title = item.Task.Title()
		description = "This will also remove its dependencies, links, and logs."
		t.deleteTarget = "task"
	case focusNotes:
		item, ok := t.noteList.SelectedItem().(NoteItem)
		if !ok {
			return t, nil
		}
		title = item.Note.Filename
		description = "This will permanently delete the note."
		t.deleteTarget = "note"
	default:
		return t, nil
	}

	t.confirmResult = new(bool)
	t.confirmForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Delete \""+title+"\"?").
				Description(description).
				Affirmative("Yes").
				Negative("No").
				Value(t.confirmResult),
		),
	).WithWidth(dialogWidth)
	return t, t.confirmForm.Init()
}

func (t *Tui) buildTaskForm() *huh.Form {
	labelOptions := make([]huh.Option[string], len(task.ValidLabels))
	for i, l := range task.ValidLabels {
		labelOptions[i] = huh.NewOption(l, l)
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				CharLimit(50).
				Value(&t.formTitle).
				Validate(huh.ValidateNotEmpty()),
			huh.NewText().
				Title("Description").
				Value(&t.formDesc).
				Lines(4),
			huh.NewInput().
				Title("Link").
				Value(&t.formLink).
				Validate(func(s string) error {
					normalized := task.NormalizeLink(s)
					return task.ValidateLink(normalized)
				}),
			huh.NewSelect[string]().
				Title("Label").
				Options(labelOptions...).
				Value(&t.formLabel),
			huh.NewSelect[int]().
				Title("Priority").
				Options(
					huh.NewOption("High (1)", 1),
					huh.NewOption("Medium (2)", 2),
					huh.NewOption("Low (3)", 3),
				).
				Value(&t.formPriority),
		),
	).WithWidth(dialogWidth).WithShowHelp(true)
}

func (t *Tui) startCreate() (tea.Model, tea.Cmd) {
	t.formTaskID = ""
	t.formTitle = ""
	t.formDesc = ""
	t.formLink = ""
	t.formLabel = "task"
	t.formPriority = 2

	t.taskForm = t.buildTaskForm()
	return t, t.taskForm.Init()
}

func (t *Tui) editTarget() int {
	if t.focus == focusDetail {
		return t.lastListFocus
	}
	return t.focus
}

func (t *Tui) startEdit() (tea.Model, tea.Cmd) {
	target := t.editTarget()
	if target == focusNotes {
		return t.startNoteEdit()
	}
	if target != focusTasks {
		return t, nil
	}
	item, ok := t.taskList.SelectedItem().(TaskItem)
	if !ok {
		return t, nil
	}

	tk := item.Task
	t.formTaskID = tk.ID()
	t.formTitle = tk.Title()
	t.formDesc = tk.Description()
	t.formLink = tk.Link()
	t.formLabel = tk.Label()
	if t.formLabel == "" {
		t.formLabel = "task"
	}
	t.formPriority = tk.Priority()
	if t.formPriority == 0 {
		t.formPriority = 2
	}

	t.taskForm = t.buildTaskForm()
	return t, t.taskForm.Init()
}

// taskFormSaveCmd persists the created or edited task and reloads data.
func (t *Tui) taskFormSaveCmd() tea.Cmd {
	taskID := t.formTaskID
	title := t.formTitle
	desc := t.formDesc
	link := t.formLink
	label := t.formLabel
	priority := t.formPriority
	taskSvc := t.taskService
	noteSvc := t.noteService

	if taskID == "" {
		// Create
		return func() tea.Msg {
			id := taskSvc.GenerateTaskID()
			newTask := task.NewTaskComplete(id, task.Todo, title, desc, priority, link)
			newTask.SetLabel(label)
			taskSvc.CreateTask(newTask)
			taskSvc.SetLabel(id, label)
			return fetchData(taskSvc, noteSvc)
		}
	}

	// Edit
	return func() tea.Msg {
		tk, err := taskSvc.GetTaskByID(taskID)
		if err != nil {
			return fetchData(taskSvc, noteSvc)
		}
		updated := task.NewTaskComplete(tk.ID(), tk.Status(), title, desc, priority, link)
		updated.SetLabel(label)
		updated.SetBlockedBy(tk.BlockedBy())
		updated.SetBlocks(tk.Blocks())
		updated.SetNotes(tk.Notes())
		taskSvc.UpdateTask(updated)
		taskSvc.SetLabel(taskID, label)
		return fetchData(taskSvc, noteSvc)
	}
}

// editorFinishedMsg is sent when an external editor process exits.
type editorFinishedMsg struct{ err error }

// resolveEditor returns the first available editor, preferring nvim.
func resolveEditor() string {
	candidates := []string{"nvim"}
	if e := os.Getenv("EDITOR"); e != "" {
		candidates = append(candidates, e)
	}
	candidates = append(candidates, "vim", "vi", "nano")
	for _, e := range candidates {
		if _, err := exec.LookPath(e); err == nil {
			return e
		}
	}
	return "vi"
}

func (t *Tui) buildNoteForm() *huh.Form {
	noteSvc := t.noteService
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Filename").
				Description("Without .md extension").
				Value(&t.formNoteFilename).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("filename is required")
					}
					if strings.Contains(s, "/") || strings.Contains(s, "\\") || strings.HasPrefix(s, ".") {
						return fmt.Errorf("filename must not contain path separators or start with a dot")
					}
					path := noteSvc.GetNotePath(s)
					if _, err := os.Stat(path); err == nil {
						return fmt.Errorf("note %s already exists", s)
					} else if !os.IsNotExist(err) {
						return fmt.Errorf("unable to check note %s: %v", s, err)
					}
					return nil
				}),
		),
	).WithWidth(dialogWidth).WithShowHelp(true)
}

func (t *Tui) startNoteCreate() (tea.Model, tea.Cmd) {
	t.formNoteFilename = ""
	t.noteForm = t.buildNoteForm()
	return t, t.noteForm.Init()
}

// noteFormOpenEditorCmd writes a template note and opens it in an editor.
func (t *Tui) noteFormOpenEditorCmd() tea.Cmd {
	filename := t.formNoteFilename
	noteSvc := t.noteService
	t.formNoteFilename = ""

	editor := resolveEditor()
	path := noteSvc.GetNotePath(filename)

	// Track the filename so we can select it after data reload.
	if !strings.HasSuffix(filename, ".md") {
		t.pendingNoteSelect = filename + ".md"
	} else {
		t.pendingNoteSelect = filename
	}

	// Write template inside the Cmd closure (TEA: no I/O in Update),
	// then return the exec message to launch the editor.
	return func() tea.Msg {
		if err := noteSvc.WriteNote(filename, note.DefaultTemplate(filename)); err != nil {
			return editorFinishedMsg{err}
		}
		c := exec.Command(editor, path)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return editorFinishedMsg{err}
		})()
	}
}

func (t *Tui) startNoteEdit() (tea.Model, tea.Cmd) {
	item, ok := t.noteList.SelectedItem().(NoteItem)
	if !ok {
		return t, nil
	}
	editor := resolveEditor()
	filename := strings.TrimSuffix(item.Note.Filename, ".md")
	path := t.noteService.GetNotePath(filename)
	c := exec.Command(editor, path)
	return t, tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})
}

// openLinkCmd opens the selected task's link in the default browser.
func (t *Tui) openLinkCmd() tea.Cmd {
	target := t.editTarget()
	if target != focusTasks {
		return nil
	}
	item, ok := t.taskList.SelectedItem().(TaskItem)
	if !ok {
		return nil
	}
	link := item.Task.Link()
	if link == "" {
		return nil
	}
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", link)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", link)
		default:
			cmd = exec.Command("xdg-open", link)
		}
		cmd.Start()
		return nil
	}
}

// taskStatusUpdatedMsg signals that a task's status was persisted.
type taskStatusUpdatedMsg struct {
	task task.Task
}

// cycleTaskStatusCmd cycles the selected task's status forward or backward.
// It returns a Cmd that persists the change and sends a taskStatusUpdatedMsg.
// When cycling to Done, it uses CloseTask to remove blocking dependencies
// (matching the CLI's "pace task close" behavior), then triggers a full data
// reload so unblocked tasks update in the list.
func (t *Tui) cycleTaskStatusCmd(forward bool) tea.Cmd {
	item, ok := t.taskList.SelectedItem().(TaskItem)
	if !ok {
		return nil
	}
	tk := item.Task
	if tk.IsBlocked() {
		return nil
	}
	var next task.Status
	if forward {
		next = tk.Status().GetNext()
	} else {
		next = tk.Status().GetPrev()
	}
	if err := tk.SetStatus(next); err != nil {
		return nil
	}
	taskSvc := t.taskService
	noteSvc := t.noteService
	return func() tea.Msg {
		if next == task.Done {
			if err := taskSvc.CloseTask(tk.ID(), ""); err != nil {
				return fetchData(taskSvc, noteSvc)
			}
			// Deps changed — reload everything so blocked tasks update.
			return fetchData(taskSvc, noteSvc)
		}
		if err := taskSvc.UpdateTask(tk); err != nil {
			return fetchData(taskSvc, noteSvc)
		}
		return taskStatusUpdatedMsg{task: tk}
	}
}

// dataReloadedMsg carries refreshed list items after a mutation or initial load.
type dataReloadedMsg struct {
	tasks []list.Item
	notes []list.Item
}

// detailRenderedMsg carries pre-rendered detail content for the viewport.
type detailRenderedMsg struct {
	seq     uint64
	key     string
	content string
}

func sortTasks(tasks []task.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		order := func(s task.Status) int {
			switch s {
			case task.InProgress:
				return 0
			case task.Todo:
				return 1
			case task.Done:
				return 2
			default:
				return 3
			}
		}
		oi, oj := order(tasks[i].Status()), order(tasks[j].Status())
		if oi != oj {
			return oi < oj
		}
		return tasks[i].Priority() < tasks[j].Priority()
	})
}

func fetchData(taskSvc *task.Service, noteSvc *note.Service) dataReloadedMsg {
	var taskItems []list.Item
	if tasks, err := taskSvc.LoadAllTasks(); err == nil {
		sortTasks(tasks)
		taskItems = make([]list.Item, len(tasks))
		for i, tk := range tasks {
			taskItems[i] = TaskItem{Task: tk}
		}
	}

	var noteItems []list.Item
	if notes, err := noteSvc.ListNotesWithMeta(false); err == nil {
		noteItems = make([]list.Item, len(notes))
		for i, n := range notes {
			noteItems[i] = NoteItem{Note: n}
		}
	}

	return dataReloadedMsg{tasks: taskItems, notes: noteItems}
}

// loadDataCmd returns a tea.Cmd that loads all data from services.
func (t *Tui) loadDataCmd() tea.Cmd {
	taskSvc := t.taskService
	noteSvc := t.noteService
	return func() tea.Msg {
		return fetchData(taskSvc, noteSvc)
	}
}

// deleteCmd returns a tea.Cmd that performs the deletion and reloads data.
func (t *Tui) deleteCmd(target string) tea.Cmd {
	var taskID string
	var noteFilename string
	switch target {
	case "task":
		if item, ok := t.taskList.SelectedItem().(TaskItem); ok {
			taskID = item.Task.ID()
		}
	case "note":
		if item, ok := t.noteList.SelectedItem().(NoteItem); ok {
			noteFilename = item.Note.Filename
		}
	}

	taskSvc := t.taskService
	noteSvc := t.noteService

	return func() tea.Msg {
		switch target {
		case "task":
			if taskID != "" {
				taskSvc.DeleteTask(taskID)
			}
		case "note":
			if noteFilename != "" {
				noteSvc.DeleteNote(noteFilename)
			}
		}
		return fetchData(taskSvc, noteSvc)
	}
}

func (t *Tui) updateFilteringList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if t.taskList.FilterState() == list.Filtering {
		t.taskList, cmd = t.taskList.Update(msg)
	} else {
		t.noteList, cmd = t.noteList.Update(msg)
	}
	return t, cmd
}

func (t *Tui) updateViewport(msg tea.Msg) (tea.Model, tea.Cmd) {
	kmsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return t, nil
	}
	switch kmsg.String() {
	case "j", "down":
		t.viewport.LineDown(1)
	case "k", "up":
		t.viewport.LineUp(1)
	case "g", "home":
		t.viewport.GotoTop()
	case "G", "end":
		t.viewport.GotoBottom()
	case "u", "ctrl+u":
		t.viewport.HalfPageUp()
	case "d", "ctrl+d":
		t.viewport.HalfPageDown()
	default:
		var cmd tea.Cmd
		t.viewport, cmd = t.viewport.Update(msg)
		return t, cmd
	}
	return t, nil
}

func (t *Tui) refreshDetailCmd() tea.Cmd {
	var itemKey string
	var tk *task.Task
	var nt *note.Note

	// When already in the detail panel (e.g. data reload), re-render whichever
	// list last provided the detail content.
	listFocus := t.focus
	if listFocus == focusDetail {
		switch {
		case strings.HasPrefix(t.lastKey, "task:"):
			listFocus = focusTasks
		case strings.HasPrefix(t.lastKey, "note:"):
			listFocus = focusNotes
		default:
			// If lastKey was cleared before this refresh, fall back to the last
			// list focus so we preserve the previous detail source.
			if t.lastListFocus == focusTasks || t.lastListFocus == focusNotes {
				listFocus = t.lastListFocus
			} else {
				listFocus = focusTasks
			}
		}
	}

	switch listFocus {
	case focusTasks:
		if item, ok := t.taskList.SelectedItem().(TaskItem); ok {
			itemKey = "task:" + item.Task.ID()
			cp := item.Task
			tk = &cp
		}
	case focusNotes:
		if item, ok := t.noteList.SelectedItem().(NoteItem); ok {
			itemKey = "note:" + item.Note.Filename
			cp := item.Note
			nt = &cp
		}
	}

	if itemKey == "" {
		t.lastKey = ""
		t.viewport.SetContent(detailPlaceholder)
		return nil
	}
	if itemKey == t.lastKey {
		return nil
	}

	// Clear stale content immediately so the View shows the spinner
	// while the async render is in flight, instead of the previous item.
	t.lastKey = ""
	t.detailLoading = true

	t.detailSeq++
	seq := t.detailSeq
	w := t.contentWidth()
	taskSvc := t.taskService
	noteSvc := t.noteService

	renderCmd := func() tea.Msg {
		var content string
		if tk != nil {
			content = renderTaskDetail(*tk, w, taskSvc)
		} else if nt != nil {
			content = renderNoteDetail(*nt, w, noteSvc)
		}
		return detailRenderedMsg{seq: seq, key: itemKey, content: content}
	}

	return tea.Batch(renderCmd, t.spinner.Tick)
}

// renderFieldBox builds a bordered box for a single field:
//
//	┌ Label ─────┐
//	│ value      │
//	└────────────┘
func renderFieldBox(label, value string, valueStyle lipgloss.Style, boxWidth int) string {
	innerW := boxWidth - 2 // subtract left+right border chars

	// Top border: ┌ Label ───┐
	topLabel := " " + label + " "
	fillLen := innerW - len(topLabel)
	if fillLen < 0 {
		fillLen = 0
	}
	top := fieldBoxBorder.Render("┌") + fieldBoxLabel.Render(topLabel) + fieldBoxBorder.Render(strings.Repeat("─", fillLen)+"┐")

	// Middle: │ styled_value   │
	styledVal := valueStyle.Render(value)
	valVisualW := lipgloss.Width(styledVal)
	padLen := innerW - 1 - valVisualW // 1 for leading space
	if padLen < 0 {
		padLen = 0
	}
	mid := fieldBoxBorder.Render("│") + " " + styledVal + strings.Repeat(" ", padLen) + fieldBoxBorder.Render("│")

	// Bottom: └────────────┘
	bot := fieldBoxBorder.Render("└" + strings.Repeat("─", innerW) + "┘")

	return top + "\n" + mid + "\n" + bot
}

// renderFieldBoxes renders the Status, Priority, and Label boxes side-by-side.
func renderFieldBoxes(tk task.Task, w int) string {
	const boxW = 14
	const gap = 2

	// Status
	var statusText string
	var statusStyle lipgloss.Style
	if tk.IsBlocked() {
		statusText = "⊘ blocked"
		statusStyle = statusBlocked
	} else {
		switch tk.Status() {
		case task.Todo:
			statusText = "○ todo"
			statusStyle = statusTodo
		case task.InProgress:
			statusText = "● active"
			statusStyle = statusProgress
		case task.Done:
			statusText = "● done"
			statusStyle = statusDone
		default:
			statusText = tk.Status().String()
			statusStyle = detailValue
		}
	}

	// Priority
	var priText string
	var priStyle lipgloss.Style
	switch tk.Priority() {
	case 1:
		priText = "! high"
		priStyle = priorityP1
	case 2:
		priText = "~ medium"
		priStyle = priorityP2
	case 3:
		priText = "· low"
		priStyle = priorityP3
	default:
		priText = task.PriorityName(tk.Priority())
		priStyle = detailValue
	}

	// Label
	lbl := tk.Label()
	if lbl == "" {
		lbl = "task"
	}

	box1 := renderFieldBox("Status", statusText, statusStyle, boxW)
	box2 := renderFieldBox("Priority", priText, priStyle, boxW)
	box3 := renderFieldBox("Label", lbl, detailValue, boxW)

	if w < 46 {
		return box1 + "\n" + box2 + "\n" + box3
	}

	spacer := strings.Repeat(" ", gap)
	return lipgloss.JoinHorizontal(lipgloss.Top, box1, spacer, box2, spacer, box3)
}

// renderChips renders items as [item1] [item2] with styled brackets.
func renderChips(items []string) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = chipBracket.Render("[") + chipText.Render(item) + chipBracket.Render("]")
	}
	return strings.Join(parts, " ")
}

func renderTaskDetail(tk task.Task, w int, taskSvc *task.Service) string {
	// wrapStyled wraps plain text to width, then applies a style to each line
	// so ANSI codes don't interfere with the wrap calculation.
	wrapStyled := func(s string, style lipgloss.Style) string {
		lines := strings.Split(wrap.String(s, w), "\n")
		for i, line := range lines {
			lines[i] = style.Render(line)
		}
		return strings.Join(lines, "\n")
	}

	// wrapIndent wraps text at width-indent, then prefixes continuation lines
	// with indent spaces so they align under the first line's content.
	wrapIndent := func(s string, indent int, style lipgloss.Style) string {
		iw := w - indent
		if iw < 10 {
			iw = 10
		}
		lines := strings.Split(wrap.String(s, iw), "\n")
		pad := strings.Repeat(" ", indent)
		for i, line := range lines {
			if i > 0 {
				lines[i] = pad + style.Render(line)
			} else {
				lines[i] = style.Render(line)
			}
		}
		return strings.Join(lines, "\n")
	}

	var b strings.Builder

	// Title
	b.WriteString(wrapStyled(tk.Title(), detailTitle))
	b.WriteString("\n")

	// ID
	b.WriteString(detailID.Render(tk.ID()))
	b.WriteString("\n\n")

	// Field boxes (Status, Priority, Label)
	b.WriteString(renderFieldBoxes(tk, w))
	b.WriteString("\n")

	// Description section
	b.WriteString("\n")
	b.WriteString(detailSection.Render("Description"))
	b.WriteString("\n\n")
	if desc := tk.Description(); desc != "" {
		b.WriteString(wrapStyled(desc, detailDesc))
	} else {
		b.WriteString(detailDim.Render("No description"))
	}
	b.WriteString("\n")

	// Metadata section
	b.WriteString("\n")
	b.WriteString(detailSection.Render("Metadata"))
	b.WriteString("\n\n")

	hasLink := tk.Link() != ""
	hasBlockedBy := len(tk.BlockedBy()) > 0
	hasBlocks := len(tk.Blocks()) > 0
	hasNotes := len(tk.Notes()) > 0
	if hasLink || hasBlockedBy || hasBlocks || hasNotes {
		const metaLabelW = 13
		metaRow := func(label, value string) {
			padded := label + ":" + strings.Repeat(" ", metaLabelW-len(label)-1)
			b.WriteString(detailLinkLabel.Render(padded))
			b.WriteString(value)
			b.WriteString("\n")
		}

		if hasLink {
			metaRow("Link", wrapIndent(tk.Link(), metaLabelW, detailLinkURL))
		}
		if hasBlockedBy {
			metaRow("Blocked by", renderChips(tk.BlockedBy()))
		}
		if hasBlocks {
			metaRow("Blocks", renderChips(tk.Blocks()))
		}
		if hasNotes {
			metaRow("Notes", renderChips(tk.Notes()))
		}
	} else {
		b.WriteString(detailDim.Render("No metadata"))
		b.WriteString("\n")
	}

	// Logs
	type logEntry struct {
		time      string
		isOutcome bool
		message   string
	}
	var logEntries []logEntry
	if taskSvc != nil {
		if logs, err := taskSvc.GetTaskLogs(tk.ID()); err == nil && len(logs) > 0 {
			for _, l := range logs {
				logEntries = append(logEntries, logEntry{
					time:      l.CreatedAt,
					isOutcome: l.Type == "outcome",
					message:   l.Message,
				})
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(detailLabel.Render(fmt.Sprintf("Logs (%d):", len(logEntries))))
	b.WriteString("\n")
	for _, entry := range logEntries {
		styledPrefix := detailLogTime.Render(entry.time + " ")
		prefixW := len(entry.time) + 1
		if entry.isOutcome {
			styledPrefix += detailOutcome.Render("[outcome] ")
			prefixW += len("[outcome] ")
		}
		indent := 2 + prefixW
		b.WriteString("  ")
		b.WriteString(styledPrefix)
		b.WriteString(wrapIndent(entry.message, indent, detailValue))
		b.WriteString("\n")
	}

	return b.String()
}


func renderNoteDetail(n note.Note, w int, noteSvc *note.Service) string {
	// Lazy-load full note metadata (content, tasks, labels) when viewing detail.
	if noteSvc != nil {
		if full, err := noteSvc.ReadNoteWithMeta(n.Filename); err == nil {
			n = *full
		}
	}

	var b strings.Builder
	b.WriteString(detailHeader.Render(wrap.String(n.Filename, w)))
	b.WriteString("\n")

	if n.Description != "" {
		b.WriteString(detailLabel.Render(wrap.String(n.Description, w)))
		b.WriteString("\n\n")
	}

	if len(n.Tasks) > 0 {
		b.WriteString(detailLabel.Render("Linked tasks: "))
		b.WriteString(detailValue.Render(wrap.String(strings.Join(n.Tasks, ", "), w)))
		b.WriteString("\n\n")
	}

	if n.Content != "" {
		b.WriteString(note.RenderMarkdownWithWidth(n.Content, w))
	}

	return b.String()
}

func (t *Tui) listWidth() int {
	w := int(float64(t.width) * 0.35)
	if w < minListW {
		w = minListW
	}
	if w > t.width-minListW {
		w = t.width - minListW
	}
	return w
}

func (t *Tui) detailWidth() int {
	return t.width - t.listWidth()
}

// contentWidth returns the usable character width inside the detail viewport,
// accounting for the border (2) and some inner padding (2).
func (t *Tui) contentWidth() int {
	w := t.detailWidth() - 4
	if w < 20 {
		w = 20
	}
	return w
}

// fitHeight ensures s is exactly h lines tall, truncating or padding as needed.
func fitHeight(s string, h int) string {
	if h <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (t *Tui) recalcLayout() {
	// Cache help height — only recompute when terminal width changes.
	helpH := t.cachedHelpH
	if t.width != t.cachedHelpWidth {
		t.help.Width = t.width - 2 // account for helpStyle horizontal padding
		helpH = lipgloss.Height(t.help.View(tuiKeys))
		t.cachedHelpH = helpH
		t.cachedHelpWidth = t.width
	}

	availH := t.height - helpH
	lw := t.listWidth() - 2
	dw := t.detailWidth() - 2

	listH := availH - overviewH
	taskH := listH / 2
	noteH := listH - taskH

	t.taskList.SetSize(lw, taskH-2)
	t.noteList.SetSize(lw, noteH-2)
	t.viewport.Width = dw
	t.viewport.Height = availH - 2

	t.layoutAvailH = availH
	t.layoutTaskH = taskH
	t.layoutNoteH = noteH
}

func shortenPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func (t *Tui) renderOverview(w int) string {
	var total, todo, inProg, done int
	for _, item := range t.taskList.Items() {
		if ti, ok := item.(TaskItem); ok {
			total++
			switch ti.Task.Status() {
			case task.Todo:
				todo++
			case task.InProgress:
				inProg++
			case task.Done:
				done++
			}
		}
	}

	// Storage path with type tag
	tag := storageTag.Render("[" + string(t.storageType) + "]")
	storeLine := overviewLabel.Render(shortenPath(t.storagePath)) + "  " + tag

	if total == 0 {
		content := "\n" + storeLine + "\n\n" + noTasksStyle.Render("no tasks yet")
		return fitHeight(content, overviewH)
	}

	// Progress bar with percentage
	pct := done * 100 / total
	pctStr := overviewPct.Render(fmt.Sprintf(" %d%%", pct))
	barW := w - 6 // room for " XX%"
	if barW < 10 {
		barW = 10
	}
	filled := done * barW / total
	bar := progressFilled.Render(strings.Repeat("━", filled)) +
		progressEmpty.Render(strings.Repeat("─", barW-filled)) +
		pctStr

	// Counts line: done/total + status breakdown
	counts := overviewLabel.Render(fmt.Sprintf("Progress %d/%d  ", done, total)) +
		statusTodo.Render(fmt.Sprintf("○ %d", todo)) +
		overviewLabel.Render("  ") +
		statusProgress.Render(fmt.Sprintf("● %d", inProg)) +
		overviewLabel.Render("  ") +
		statusDone.Render(fmt.Sprintf("✓ %d", done))

	content := "\n" + storeLine + "\n\n" + bar + "\n\n" + counts
	return fitHeight(content, overviewH)
}

func (t *Tui) View() string {
	if t.quitting {
		return ""
	}
	if !t.loaded {
		return "Loading..."
	}
	if t.tooSmall {
		msg := fmt.Sprintf("Terminal too small (%dx%d).\nMinimum size: %dx%d.\nPlease resize your terminal.", t.width, t.height, minWidth, minHeight)
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(1, 2)
		return style.Render(msg)
	}

	lw := t.listWidth()
	dw := t.detailWidth()
	availH := t.layoutAvailH
	taskH := t.layoutTaskH
	noteH := t.layoutNoteH

	bdr := func(focused bool) lipgloss.Style {
		if focused {
			return focusedBorder
		}
		return blurredBorder
	}

	overview := lipgloss.NewStyle().Width(lw).PaddingLeft(1).Render(t.renderOverview(lw))

	var taskContent string
	if len(t.taskList.Items()) == 0 {
		title := listTitleStyle.Render(t.taskList.Title)
		taskContent = fitHeight(title+"\n\n"+noTasksStyle.Render("  press + to create a task"), taskH-2)
	} else {
		taskContent = fitHeight(t.taskList.View(), taskH-2)
	}
	taskBox := bdr(t.focus == focusTasks).Width(lw - 2).Render(taskContent)

	var noteContent string
	if len(t.noteList.Items()) == 0 {
		title := listTitleStyle.Render(t.noteList.Title)
		noteContent = fitHeight(title+"\n\n"+noTasksStyle.Render("  press + to create a note"), noteH-2)
	} else {
		noteContent = fitHeight(t.noteList.View(), noteH-2)
	}
	noteBox := bdr(t.focus == focusNotes).Width(lw - 2).Render(noteContent)

	var detailContent string
	if t.detailLoading {
		detailContent = lipgloss.Place(dw-2, availH-2, lipgloss.Center, lipgloss.Center,
			detailDim.Render(t.spinner.View()+" Loading..."))
	} else if t.lastKey == "" {
		detailContent = lipgloss.Place(dw-2, availH-2, lipgloss.Center, lipgloss.Center,
			detailDim.Render(detailPlaceholder))
	} else {
		detailContent = fitHeight(t.viewport.View(), availH-2)
	}
	detailBox := bdr(t.focus == focusDetail).Width(dw - 2).Render(detailContent)

	left := lipgloss.JoinVertical(lipgloss.Left, overview, taskBox, noteBox)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, detailBox)
	footer := helpStyle.Render(t.help.View(tuiKeys))

	view := lipgloss.JoinVertical(lipgloss.Left, panels, footer)

	if t.confirmForm != nil {
		dialog := lipgloss.NewStyle().Width(dialogWidth).Render(t.confirmForm.View())
		return lipgloss.Place(
			t.width,
			t.height,
			lipgloss.Center,
			lipgloss.Center,
			dialog,
		)
	}

	if t.taskForm != nil {
		dialog := lipgloss.NewStyle().Width(dialogWidth).Render(t.taskForm.View())
		return lipgloss.Place(
			t.width,
			t.height,
			lipgloss.Center,
			lipgloss.Center,
			dialog,
		)
	}

	if t.noteForm != nil {
		dialog := lipgloss.NewStyle().Width(dialogWidth).Render(t.noteForm.View())
		return lipgloss.Place(
			t.width,
			t.height,
			lipgloss.Center,
			lipgloss.Center,
			dialog,
		)
	}

	return view
}
