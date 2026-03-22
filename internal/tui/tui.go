package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"

	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/task"
)

const (
	focusTasks   = 0
	focusNotes   = 1
	focusDetail  = 2
	minListW     = 30
	minWidth     = 80
	minHeight    = 20
	dialogWidth  = 50
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
	listTitleBarStyle = lipgloss.NewStyle()

	// Detail panel styles
	detailHeader   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)
	detailTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	detailID       = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	detailDesc     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	detailDescNone = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	detailLabel    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	detailValue    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	detailDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	detailSection  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	detailLogTime  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	detailOutcome  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusTodo     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	statusProgress = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	statusDone     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusBlocked  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	priorityP1     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	priorityP2     = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	priorityP3     = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	priorityP4     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

type Tui struct {
	taskList    list.Model
	noteList    list.Model
	viewport    viewport.Model
	help        help.Model
	taskService *task.Service
	noteService *note.Service
	focus       int
	width       int
	height      int
	loaded      bool
	quitting    bool
	tooSmall    bool
	lastKey       string
	detailSeq     uint64
	confirmForm   *huh.Form
	confirmResult *bool
	deleteTarget  string // "task" or "note"
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

	noteList := newList("Notes", nil, noteDelegate{})
	noteList.SetFilteringEnabled(false)

	return &Tui{
		taskList:    newList("Tasks", nil, taskDelegate{}),
		noteList:    noteList,
		viewport:    viewport.New(0, 0),
		help:        help.New(),
		taskService: taskService,
		noteService: noteService,
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

	switch msg := msg.(type) {
	case dataReloadedMsg:
		noteIdx := t.noteList.Index()
		t.taskList.SetItems(msg.tasks)
		t.noteList.SetItems(msg.notes)
		if noteIdx >= len(msg.notes) && len(msg.notes) > 0 {
			t.noteList.Select(len(msg.notes) - 1)
		}
		t.lastKey = ""
		return t, t.refreshDetailCmd()

	case detailRenderedMsg:
		if msg.seq != t.detailSeq {
			return t, nil // stale render from an older request; discard
		}
		t.lastKey = msg.key
		t.viewport.SetContent(msg.content)
		t.viewport.GotoTop()
		return t, nil

	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		t.tooSmall = msg.Width < minWidth || msg.Height < minHeight
		if !t.tooSmall {
			t.recalcLayout()
		}
		t.loaded = true
		t.lastKey = "" // force detail re-render at new width
		if !t.tooSmall {
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

		case key.Matches(msg, tuiKeys.Delete):
			return t.startDelete()

		case key.Matches(msg, tuiKeys.Tab):
			if t.focus == focusTasks {
				t.focus = focusNotes
			} else {
				t.focus = focusTasks
			}
			t.syncFilterEnabled()
			return t, t.refreshDetailCmd()

		case key.Matches(msg, tuiKeys.Right):
			if t.focus != focusDetail {
				t.focus = focusDetail
				return t, nil
			}

		case key.Matches(msg, tuiKeys.Left):
			if t.focus == focusDetail {
				t.focus = focusTasks
				t.syncFilterEnabled()
				return t, t.refreshDetailCmd()
			}
		}

		switch t.focus {
		case focusTasks:
			var cmd tea.Cmd
			t.taskList, cmd = t.taskList.Update(msg)
			return t, tea.Batch(cmd, t.refreshDetailCmd())
		case focusNotes:
			var cmd tea.Cmd
			t.noteList, cmd = t.noteList.Update(msg)
			return t, tea.Batch(cmd, t.refreshDetailCmd())
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
	// Only refresh detail when filtering completes, not on every keystroke.
	// The isFiltering() check after Update sees the new state — if the user
	// just pressed Enter/Esc, filtering has ended and we should update.
	if !t.isFiltering() {
		return t, tea.Batch(cmd, t.refreshDetailCmd())
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
	if t.focus == focusDetail {
		return nil
	}

	var itemKey string
	var tk *task.Task
	var nt *note.Note

	switch t.focus {
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
		t.viewport.SetContent("  Select an item to view details")
		return nil
	}
	if itemKey == t.lastKey {
		return nil
	}

	t.detailSeq++
	seq := t.detailSeq
	w := t.contentWidth()
	taskSvc := t.taskService
	noteSvc := t.noteService

	return func() tea.Msg {
		var content string
		if tk != nil {
			content = renderTaskDetail(*tk, w, taskSvc)
		} else if nt != nil {
			content = renderNoteDetail(*nt, w, noteSvc)
		}
		return detailRenderedMsg{seq: seq, key: itemKey, content: content}
	}
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

	// --- Title (prominent) ---
	b.WriteString(wrapStyled(tk.Title(), detailTitle))
	b.WriteString("\n")

	// --- ID ---
	b.WriteString(detailID.Render(tk.ID()))
	b.WriteString("\n\n")

	// --- Status · Priority · Label on one line ---
	label := tk.Label()
	if label == "" {
		label = "task"
	}
	dot := detailDim.Render(" · ")
	meta := renderStatus(tk) + dot + renderPriority(tk) + dot + detailValue.Render(label)
	b.WriteString(meta)
	b.WriteString("\n\n")

	// --- Description section ---
	b.WriteString(detailSection.Render("─── Description"))
	b.WriteString("\n\n")
	if desc := tk.Description(); desc != "" {
		b.WriteString(wrapStyled(desc, detailDesc))
	} else {
		b.WriteString(detailDescNone.Render("No description"))
	}
	b.WriteString("\n")

	// --- Metadata section ---
	hasMetadata := tk.Link() != "" || len(tk.BlockedBy()) > 0 || len(tk.Blocks()) > 0 || len(tk.Notes()) > 0

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
			hasMetadata = true
		}
	}

	if hasMetadata {
		b.WriteString("\n")
		b.WriteString(detailSection.Render("─── Metadata"))
		b.WriteString("\n\n")

		const labelW = 12 // "Blocked by  " width for alignment
		if tk.Link() != "" {
			b.WriteString(detailLabel.Render("Link        "))
			b.WriteString(wrapIndent(tk.Link(), labelW, detailValue))
			b.WriteString("\n")
		}
		if len(tk.BlockedBy()) > 0 {
			b.WriteString(detailLabel.Render("Blocked by  "))
			b.WriteString(wrapIndent(strings.Join(tk.BlockedBy(), ", "), labelW, detailValue))
			b.WriteString("\n")
		}
		if len(tk.Blocks()) > 0 {
			b.WriteString(detailLabel.Render("Blocks      "))
			b.WriteString(wrapIndent(strings.Join(tk.Blocks(), ", "), labelW, detailValue))
			b.WriteString("\n")
		}
		if len(tk.Notes()) > 0 {
			b.WriteString(detailLabel.Render("Notes       "))
			b.WriteString(wrapIndent(strings.Join(tk.Notes(), ", "), labelW, detailValue))
			b.WriteString("\n")
		}

		if len(logEntries) > 0 {
			if len(tk.BlockedBy()) > 0 || len(tk.Blocks()) > 0 || len(tk.Notes()) > 0 {
				b.WriteString("\n")
			}
			b.WriteString(detailLabel.Render(fmt.Sprintf("Logs (%d):", len(logEntries))))
			b.WriteString("\n")
			for _, entry := range logEntries {
				styledPrefix := detailLogTime.Render(entry.time + " ")
				prefixW := len(entry.time) + 1
				if entry.isOutcome {
					styledPrefix += detailOutcome.Render("[outcome] ")
					prefixW += len("[outcome] ")
				}
				indent := 2 + prefixW // 2 for list indent
				b.WriteString("  ")
				b.WriteString(styledPrefix)
				b.WriteString(wrapIndent(entry.message, indent, detailValue))
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func renderStatus(tk task.Task) string {
	if tk.IsBlocked() {
		return statusBlocked.Render("⊘ blocked")
	}
	switch tk.Status() {
	case task.Todo:
		return statusTodo.Render("○ todo")
	case task.InProgress:
		return statusProgress.Render("● in-progress")
	case task.Done:
		return statusDone.Render("● done")
	default:
		return detailValue.Render(tk.Status().String())
	}
}

func renderPriority(tk task.Task) string {
	switch tk.Priority() {
	case 1:
		return priorityP1.Render("P1 (urgent)")
	case 2:
		return priorityP2.Render("P2 (high)")
	case 3:
		return priorityP3.Render("P3 (normal)")
	case 4:
		return priorityP4.Render("P4 (low)")
	default:
		return detailValue.Render(fmt.Sprintf("P%d", tk.Priority()))
	}
}

func renderNoteDetail(n note.Note, w int, noteSvc *note.Service) string {
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

	// Lazy-load note content only when viewing the detail panel
	content := n.Content
	if content == "" && noteSvc != nil {
		if raw, err := noteSvc.ReadNote(n.Filename); err == nil {
			content = raw
		}
	}
	if content != "" {
		b.WriteString(note.RenderMarkdownWithWidth(content, w))
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

func (t *Tui) recalcLayout() {
	availH := t.height - 1 // help bar
	lw := t.listWidth() - 2
	dw := t.detailWidth() - 2

	taskH := availH / 2
	noteH := availH - taskH

	t.taskList.SetSize(lw, taskH-2)
	t.noteList.SetSize(lw, noteH-2)
	t.viewport.Width = dw
	t.viewport.Height = availH - 2
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
	availH := t.height - 1
	taskH := availH / 2
	noteH := availH - taskH

	bdr := func(focused bool) lipgloss.Style {
		if focused {
			return focusedBorder
		}
		return blurredBorder
	}

	taskBox := bdr(t.focus == focusTasks).Width(lw - 2).Height(taskH - 2).Render(t.taskList.View())
	noteBox := bdr(t.focus == focusNotes).Width(lw - 2).Height(noteH - 2).Render(t.noteList.View())
	detailBox := bdr(t.focus == focusDetail).Width(dw - 2).Height(availH - 2).Render(t.viewport.View())

	left := lipgloss.JoinVertical(lipgloss.Left, taskBox, noteBox)
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

	return view
}
