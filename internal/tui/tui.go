package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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

type Tui struct {
	taskList      list.Model
	noteList      list.Model
	viewport      viewport.Model
	help          help.Model
	taskService   *task.Service
	noteService   *note.Service
	storagePath   string
	storageType   storage.StorageType
	focus         int
	width         int
	height        int
	loaded        bool
	quitting      bool
	tooSmall      bool
	lastKey       string
	lastListFocus int
	detailSeq     uint64
	spinner       spinner.Model
	detailLoading bool
	confirmForm   *huh.Form
	confirmResult *bool
	deleteTarget  string

	taskForm     *huh.Form
	formTaskID   string
	formTitle    string
	formDesc     string
	formLink     string
	formLabel    string
	formPriority int

	noteForm          *huh.Form
	formNoteFilename  string
	pendingNoteSelect string

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

// syncFilterEnabled enables filtering only on the focused list.
func (t *Tui) syncFilterEnabled() {
	t.taskList.SetFilteringEnabled(t.focus == focusTasks)
	t.noteList.SetFilteringEnabled(t.focus == focusNotes)
}

func (t *Tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		t.width = ws.Width
		t.height = ws.Height
		t.tooSmall = ws.Width < minWidth || ws.Height < minHeight
		if !t.tooSmall {
			t.recalcLayout()
		}
		t.loaded = true
	}

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
		t.lastKey = ""
		return t, tea.Batch(tea.ClearScreen, t.loadDataCmd())

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
		if t.loaded && !t.tooSmall {
			t.recalcLayout()
		}
		hadDetail := t.lastKey != "" || t.focus == focusDetail
		t.lastKey = ""
		if hadDetail {
			return t, t.refreshDetailCmd()
		}
		return t, nil

	case taskStatusUpdatedMsg:
		for i, item := range t.taskList.Items() {
			if ti, ok := item.(TaskItem); ok && ti.Task.ID() == msg.task.ID() {
				t.taskList.SetItem(i, TaskItem{Task: msg.task})
				break
			}
		}
		hadDetail := t.lastKey != ""
		t.lastKey = ""
		if hadDetail {
			return t, t.refreshDetailCmd()
		}
		return t, nil

	case detailRenderedMsg:
		if msg.seq != t.detailSeq {
			return t, nil
		}
		t.detailLoading = false
		t.lastKey = msg.key
		t.viewport.SetContent(msg.content)
		t.viewport.GotoTop()
		return t, nil

	case spinner.TickMsg:
		if t.detailLoading {
			return t.tickSpinner(msg)
		}
		return t, nil

	case tea.WindowSizeMsg:
		t.lastKey = ""
		if !t.tooSmall && t.focus == focusDetail {
			return t, t.refreshDetailCmd()
		}
		return t, nil

	case tea.KeyMsg:
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

	var cmd tea.Cmd
	switch t.focus {
	case focusTasks:
		t.taskList, cmd = t.taskList.Update(msg)
	case focusNotes:
		t.noteList, cmd = t.noteList.Update(msg)
	}
	return t, cmd
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
		return lipgloss.Place(t.width, t.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if t.taskForm != nil {
		dialog := lipgloss.NewStyle().Width(dialogWidth).Render(t.taskForm.View())
		return lipgloss.Place(t.width, t.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if t.noteForm != nil {
		dialog := lipgloss.NewStyle().Width(dialogWidth).Render(t.noteForm.View())
		return lipgloss.Place(t.width, t.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	return view
}
