package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/lucas-tremaroli/pace/internal/task"
)

const formWidth = 60

// ConfirmFormState owns a yes/no confirmation dialog.
type ConfirmFormState struct {
	form   *huh.Form
	result bool
}

func newConfirmFormState(title, description string) *ConfirmFormState {
	s := &ConfirmFormState{}
	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Affirmative("Yes").
				Negative("No").
				Value(&s.result),
		),
	).WithWidth(formWidth)
	return s
}

// TaskFormState owns the inputs and lifecycle for the task create form.
// Edit support is a deliberate follow-up — only the create path is
// populated for now, but the struct has the shape the edit form needs.
type TaskFormState struct {
	form     *huh.Form
	id       string
	title    string
	desc     string
	link     string
	label    string
	priority int
}

func newTaskFormState() *TaskFormState {
	s := &TaskFormState{
		label:    task.LabelTask,
		priority: task.PriorityMedium,
	}
	s.form = s.build()
	return s
}

// newTaskEditFormState returns a form pre-populated from an existing task.
// Submitting it updates rather than creates because s.id is non-empty.
func newTaskEditFormState(existing task.Task) *TaskFormState {
	s := &TaskFormState{
		id:       existing.ID(),
		title:    existing.Title(),
		desc:     existing.Description(),
		link:     existing.Link(),
		label:    existing.Label(),
		priority: existing.Priority(),
	}
	if s.label == "" {
		s.label = task.LabelTask
	}
	if s.priority <= 0 {
		s.priority = task.PriorityMedium
	}
	s.form = s.build()
	return s
}

func (s *TaskFormState) build() *huh.Form {
	labelOpts := make([]huh.Option[string], len(task.ValidLabels))
	for i, l := range task.ValidLabels {
		labelOpts[i] = huh.NewOption(l, l)
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
					return task.ValidateLink(task.NormalizeLink(v))
				}),
			huh.NewSelect[string]().
				Title("Label").
				Options(labelOpts...).
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
	).WithWidth(formWidth).WithShowHelp(true)
}

// NoteFormState owns the input for the new-note dialog.
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
						return fmt.Errorf("no path separators or leading dot")
					}
					exists, err := noteExists(v)
					if err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("check %s: %v", v, err)
					}
					if exists {
						return fmt.Errorf("note %s already exists", v)
					}
					return nil
				}),
		),
	).WithWidth(formWidth).WithShowHelp(true)
	return s
}
