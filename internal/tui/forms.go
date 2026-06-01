package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/task"
)

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

func (t *Tui) startDelete() (tea.Model, tea.Cmd) {
	var title, description, target string

	switch t.focus {
	case focusTasks:
		item, ok := t.taskList.SelectedItem().(TaskItem)
		if !ok {
			return t, nil
		}
		title = item.Task.Title()
		description = "This will also remove its dependencies, links, and logs."
		target = "task"
	case focusNotes:
		item, ok := t.noteList.SelectedItem().(NoteItem)
		if !ok {
			return t, nil
		}
		title = item.Note.Filename
		description = "This will permanently delete the note."
		target = "note"
	default:
		return t, nil
	}

	t.confirm = newConfirmFormState("Delete \""+title+"\"?", description, target)
	return t, t.confirm.form.Init()
}

func (t *Tui) startCreate() (tea.Model, tea.Cmd) {
	t.taskForm = newTaskFormState(nil)
	return t, t.taskForm.form.Init()
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
	t.taskForm = newTaskFormState(&tk)
	return t, t.taskForm.form.Init()
}

// taskFormSaveCmd persists the created or edited task and reloads data.
func (t *Tui) taskFormSaveCmd() tea.Cmd {
	s := t.taskForm
	taskSvc := t.taskService
	noteSvc := t.noteService

	if s.id == "" {
		return func() tea.Msg {
			id := taskSvc.GenerateTaskID()
			newTask := task.NewTaskComplete(id, task.Todo, s.title, s.desc, s.priority, s.link)
			newTask.SetLabel(s.label)
			taskSvc.CreateTask(newTask)
			taskSvc.SetLabel(id, s.label)
			return fetchData(taskSvc, noteSvc)
		}
	}

	return func() tea.Msg {
		tk, err := taskSvc.GetTaskByID(s.id)
		if err != nil {
			return fetchData(taskSvc, noteSvc)
		}
		updated := task.NewTaskComplete(tk.ID(), tk.Status(), s.title, s.desc, s.priority, s.link)
		updated.SetLabel(s.label)
		updated.SetBlockedBy(tk.BlockedBy())
		updated.SetBlocks(tk.Blocks())
		updated.SetNotes(tk.Notes())
		taskSvc.UpdateTask(updated)
		taskSvc.SetLabel(s.id, s.label)
		return fetchData(taskSvc, noteSvc)
	}
}

func (t *Tui) startNoteCreate() (tea.Model, tea.Cmd) {
	noteSvc := t.noteService
	t.noteForm = newNoteFormState(func(name string) (bool, error) {
		path := noteSvc.GetNotePath(name)
		_, err := os.Stat(path)
		if err == nil {
			return true, nil
		}
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	})
	return t, t.noteForm.form.Init()
}

// noteFormOpenEditorCmd writes a template note and opens it in an editor.
func (t *Tui) noteFormOpenEditorCmd() tea.Cmd {
	filename := t.noteForm.filename
	noteSvc := t.noteService

	editor := resolveEditor()
	path := noteSvc.GetNotePath(filename)

	if !strings.HasSuffix(filename, ".md") {
		t.pendingNoteSelect = filename + ".md"
	} else {
		t.pendingNoteSelect = filename
	}

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
