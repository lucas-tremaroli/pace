package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/lucas-tremaroli/pace/internal/task"
)

// ConfirmFormState owns the lifecycle of a yes/no confirmation dialog.
// Lives on Tui only while the dialog is open.
type ConfirmFormState struct {
	form   *huh.Form
	result bool
	target string
}

func newConfirmFormState(title, description, target string) *ConfirmFormState {
	s := &ConfirmFormState{target: target}
	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Affirmative("Yes").
				Negative("No").
				Value(&s.result),
		),
	).WithWidth(dialogWidth)
	return s
}

// TaskFormState owns the inputs and lifecycle for the task create/edit form.
type TaskFormState struct {
	form     *huh.Form
	id       string
	title    string
	desc     string
	link     string
	label    string
	priority int
}

func newTaskFormState(existing *task.Task) *TaskFormState {
	s := &TaskFormState{
		label:    task.LabelTask,
		priority: task.PriorityLow,
	}
	if existing != nil {
		s.id = existing.ID()
		s.title = existing.Title()
		s.desc = existing.Description()
		s.link = existing.Link()
		if l := existing.Label(); l != "" {
			s.label = l
		}
		if p := existing.Priority(); p > 0 {
			s.priority = p
		} else {
			s.priority = task.PriorityMedium
		}
	}
	s.form = s.buildForm()
	return s
}

func (s *TaskFormState) buildForm() *huh.Form {
	labelOptions := make([]huh.Option[string], len(task.ValidLabels))
	for i, l := range task.ValidLabels {
		labelOptions[i] = huh.NewOption(l, l)
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				CharLimit(50).
				Value(&s.title).
				Validate(huh.ValidateNotEmpty()),
			huh.NewText().
				Title("Description").
				Value(&s.desc).
				Lines(4),
			huh.NewInput().
				Title("Link").
				Value(&s.link).
				Validate(func(v string) error {
					normalized := task.NormalizeLink(v)
					return task.ValidateLink(normalized)
				}),
			huh.NewSelect[string]().
				Title("Label").
				Options(labelOptions...).
				Value(&s.label),
			huh.NewSelect[int]().
				Title("Priority").
				Options(
					huh.NewOption("High (1)", task.PriorityHigh),
					huh.NewOption("Medium (2)", task.PriorityMedium),
					huh.NewOption("Low (3)", task.PriorityLow),
				).
				Value(&s.priority),
		),
	).WithWidth(dialogWidth).WithShowHelp(true)
}

// NoteFormState owns the input and lifecycle for the new-note dialog.
type NoteFormState struct {
	form     *huh.Form
	filename string
}

func newNoteFormState(noteExists func(string) (bool, error)) *NoteFormState {
	s := &NoteFormState{}
	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Filename").
				Description("Without .md extension").
				Value(&s.filename).
				Validate(func(v string) error {
					if v == "" {
						return fmt.Errorf("filename is required")
					}
					if strings.Contains(v, "/") || strings.Contains(v, "\\") || strings.HasPrefix(v, ".") {
						return fmt.Errorf("filename must not contain path separators or start with a dot")
					}
					exists, err := noteExists(v)
					if err != nil {
						if !os.IsNotExist(err) {
							return fmt.Errorf("unable to check note %s: %v", v, err)
						}
					}
					if exists {
						return fmt.Errorf("note %s already exists", v)
					}
					return nil
				}),
		),
	).WithWidth(dialogWidth).WithShowHelp(true)
	return s
}
