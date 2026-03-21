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
	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/task"
)

const (
	focusTasks  = 0
	focusNotes  = 1
	focusDetail = 2
	minListW    = 30
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
	detailLabel    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	detailValue    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
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
	lastKey     string
}

func newList(title string, items []list.Item) list.Model {
	d := list.NewDefaultDelegate()
	d.SetHeight(1)
	d.ShowDescription = false
	d.SetSpacing(0)

	l := list.New(items, d, 0, 0)
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

	tasks, err := taskService.LoadAllTasks()
	if err != nil {
		taskService.Close()
		noteService.Close()
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	notes, err := noteService.ListNotesWithMeta(true)
	if err != nil {
		taskService.Close()
		noteService.Close()
		return nil, fmt.Errorf("failed to load notes: %w", err)
	}

	// Sort tasks: in-progress → todo → done, then by priority
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

	taskItems := make([]list.Item, len(tasks))
	for i, t := range tasks {
		taskItems[i] = TaskItem{Task: t}
	}

	noteItems := make([]list.Item, len(notes))
	for i, n := range notes {
		noteItems[i] = NoteItem{Note: n}
	}

	return &Tui{
		taskList:    newList("Tasks", taskItems),
		noteList:    newList("Notes", noteItems),
		viewport:    viewport.New(0, 0),
		help:        help.New(),
		taskService: taskService,
		noteService: noteService,
		focus:       focusTasks,
	}, nil
}

func (t *Tui) Init() tea.Cmd {
	return nil
}

func (t *Tui) isFiltering() bool {
	return t.taskList.FilterState() == list.Filtering ||
		t.noteList.FilterState() == list.Filtering
}

func (t *Tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		t.recalcLayout()
		t.loaded = true
		t.refreshDetail()
		return t, nil

	case tea.KeyMsg:
		// When a list is filtering, all keys go to it
		if t.isFiltering() {
			return t.updateFilteringList(msg)
		}

		switch {
		case key.Matches(msg, tuiKeys.Quit):
			t.quitting = true
			return t, tea.Quit

		case key.Matches(msg, tuiKeys.Tab):
			t.focus = (t.focus + 1) % 3
			t.refreshDetail()
			return t, nil
		}

		switch t.focus {
		case focusTasks:
			var cmd tea.Cmd
			t.taskList, cmd = t.taskList.Update(msg)
			t.refreshDetail()
			return t, cmd
		case focusNotes:
			var cmd tea.Cmd
			t.noteList, cmd = t.noteList.Update(msg)
			t.refreshDetail()
			return t, cmd
		case focusDetail:
			return t.updateViewport(msg)
		}
	}

	// Forward non-key messages (blink, etc.) to both lists
	var cmds []tea.Cmd
	var cmd tea.Cmd
	t.taskList, cmd = t.taskList.Update(msg)
	cmds = append(cmds, cmd)
	t.noteList, cmd = t.noteList.Update(msg)
	cmds = append(cmds, cmd)
	return t, tea.Batch(cmds...)
}

func (t *Tui) updateFilteringList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if t.taskList.FilterState() == list.Filtering {
		t.taskList, cmd = t.taskList.Update(msg)
	} else {
		t.noteList, cmd = t.noteList.Update(msg)
	}
	t.refreshDetail()
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

func (t *Tui) refreshDetail() {
	var itemKey string
	var content string

	switch t.focus {
	case focusTasks:
		if item, ok := t.taskList.SelectedItem().(TaskItem); ok {
			itemKey = "task:" + item.Task.ID()
			if itemKey != t.lastKey {
				content = t.renderTaskDetail(item.Task)
			}
		}
	case focusNotes:
		if item, ok := t.noteList.SelectedItem().(NoteItem); ok {
			itemKey = "note:" + item.Note.Filename
			if itemKey != t.lastKey {
				content = t.renderNoteDetail(item.Note)
			}
		}
	case focusDetail:
		return
	}

	if itemKey == "" {
		t.lastKey = ""
		t.viewport.SetContent("  Select an item to view details")
		return
	}
	if itemKey != t.lastKey {
		t.lastKey = itemKey
		t.viewport.SetContent(content)
		t.viewport.GotoTop()
	}
}

func (t *Tui) renderTaskDetail(tk task.Task) string {
	var b strings.Builder

	b.WriteString(detailHeader.Render("Task Details"))
	b.WriteString("\n\n")

	b.WriteString(detailLabel.Render("ID: "))
	b.WriteString(detailValue.Render(tk.ID()))
	b.WriteString("\n\n")

	b.WriteString(detailLabel.Render("Title: "))
	b.WriteString(detailValue.Render(tk.Title()))
	b.WriteString("\n\n")

	b.WriteString(detailLabel.Render("Status: "))
	if tk.IsBlocked() {
		b.WriteString(statusBlocked.Render("⊘ blocked"))
	} else {
		switch tk.Status() {
		case task.Todo:
			b.WriteString(statusTodo.Render("○ todo"))
		case task.InProgress:
			b.WriteString(statusProgress.Render("● in-progress"))
		case task.Done:
			b.WriteString(statusDone.Render("● done"))
		}
	}
	b.WriteString("\n\n")

	b.WriteString(detailLabel.Render("Priority: "))
	switch tk.Priority() {
	case 1:
		b.WriteString(priorityP1.Render("P1 (urgent)"))
	case 2:
		b.WriteString(priorityP2.Render("P2 (high)"))
	case 3:
		b.WriteString(priorityP3.Render("P3 (normal)"))
	case 4:
		b.WriteString(priorityP4.Render("P4 (low)"))
	default:
		b.WriteString(detailValue.Render(fmt.Sprintf("P%d", tk.Priority())))
	}
	b.WriteString("\n\n")

	label := tk.Label()
	if label == "" {
		label = "task"
	}
	b.WriteString(detailLabel.Render("Label: "))
	b.WriteString(detailValue.Render(label))
	b.WriteString("\n\n")

	if desc := tk.Description(); desc != "" {
		b.WriteString(detailLabel.Render("Description:"))
		b.WriteString("\n")
		b.WriteString(detailValue.Render(desc))
		b.WriteString("\n\n")
	}

	if tk.Link() != "" {
		b.WriteString(detailLabel.Render("Link: "))
		b.WriteString(detailValue.Render(tk.Link()))
		b.WriteString("\n\n")
	}

	if len(tk.BlockedBy()) > 0 {
		b.WriteString(detailLabel.Render("Blocked by: "))
		b.WriteString(detailValue.Render(strings.Join(tk.BlockedBy(), ", ")))
		b.WriteString("\n\n")
	}
	if len(tk.Blocks()) > 0 {
		b.WriteString(detailLabel.Render("Blocks: "))
		b.WriteString(detailValue.Render(strings.Join(tk.Blocks(), ", ")))
		b.WriteString("\n\n")
	}

	if len(tk.Notes()) > 0 {
		b.WriteString(detailLabel.Render("Notes: "))
		b.WriteString(detailValue.Render(strings.Join(tk.Notes(), ", ")))
		b.WriteString("\n")
	}

	return b.String()
}

func (t *Tui) renderNoteDetail(n note.Note) string {
	dw := t.detailWidth() - 4
	if dw < 20 {
		dw = 60
	}

	var b strings.Builder
	b.WriteString(detailHeader.Render(n.Filename))
	b.WriteString("\n")

	if n.Description != "" {
		b.WriteString(detailLabel.Render(n.Description))
		b.WriteString("\n\n")
	}

	if len(n.Tasks) > 0 {
		b.WriteString(detailLabel.Render("Linked tasks: "))
		b.WriteString(detailValue.Render(strings.Join(n.Tasks, ", ")))
		b.WriteString("\n\n")
	}

	if n.Content != "" {
		b.WriteString(note.RenderMarkdownWithWidth(n.Content, dw))
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

	return lipgloss.JoinVertical(lipgloss.Left, panels, footer)
}
