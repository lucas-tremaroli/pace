package tui

import (
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/task"
)

// taskStatusUpdatedMsg signals that a task's status was persisted in-place.
type taskStatusUpdatedMsg struct {
	task task.Task
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

// cycleTaskStatusCmd cycles the selected task's status forward or backward.
// When cycling to Done, it uses CloseTask to remove blocking dependencies,
// then triggers a full reload so unblocked tasks update in the list.
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
			return fetchData(taskSvc, noteSvc)
		}
		if err := taskSvc.UpdateTask(tk); err != nil {
			return fetchData(taskSvc, noteSvc)
		}
		return taskStatusUpdatedMsg{task: tk}
	}
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

func (t *Tui) refreshDetailCmd() tea.Cmd {
	var itemKey string
	var tk *task.Task
	var nt *note.Note

	listFocus := t.focus
	if listFocus == focusDetail {
		switch {
		case strings.HasPrefix(t.lastKey, "task:"):
			listFocus = focusTasks
		case strings.HasPrefix(t.lastKey, "note:"):
			listFocus = focusNotes
		default:
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

// updateFilteringList forwards messages to whichever list is currently filtering.
func (t *Tui) updateFilteringList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if t.taskList.FilterState() == list.Filtering {
		t.taskList, cmd = t.taskList.Update(msg)
	} else {
		t.noteList, cmd = t.noteList.Update(msg)
	}
	return t, cmd
}

// updateViewport handles keyboard navigation inside the detail viewport.
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

// tickSpinner forwards spinner tick messages to keep the spinner animated.
func (t *Tui) tickSpinner(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	t.spinner, cmd = t.spinner.Update(msg)
	return t, cmd
}
