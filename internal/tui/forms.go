package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
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
	t.formPriority = 3

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
		return func() tea.Msg {
			id := taskSvc.GenerateTaskID()
			newTask := task.NewTaskComplete(id, task.Todo, title, desc, priority, link)
			newTask.SetLabel(label)
			taskSvc.CreateTask(newTask)
			taskSvc.SetLabel(id, label)
			return fetchData(taskSvc, noteSvc)
		}
	}

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
