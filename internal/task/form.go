package task

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// formField represents which field is currently focused
type formField int

const (
	fieldTitle formField = iota
	fieldDescription
	fieldLink
	fieldLabel
	fieldPriority
)

type Form struct {
	help        help.Model
	title       textinput.Model
	description textarea.Model
	link        textinput.Model
	label       TaskType
	priority    int
	col         column
	index       int
	board       *Board
	focused     formField
	isEdit      bool
}

func NewForm(title, description string, board *Board) *Form {
	form := Form{
		help:        help.New(),
		title:       textinput.New(),
		description: textarea.New(),
		link:        textinput.New(),
		label:       TypeTask,
		priority:    2,
		board:       board,
		focused:     fieldTitle,
		isEdit:      false,
	}

	form.title.Placeholder = "Task title"
	form.title.CharLimit = 50
	form.title.Width = 50
	form.description.Placeholder = "Description (optional)"
	form.description.SetHeight(5)
	form.link.Placeholder = "github.com/... (https:// added automatically)"
	form.link.CharLimit = 200
	form.link.Width = 50
	form.title.SetValue(title)
	form.description.SetValue(description)
	form.title.Focus()
	return &form
}

// NewFormWithTask creates a form pre-populated with an existing task's values
func NewFormWithTask(t Task, board *Board) *Form {
	form := NewForm(t.Title(), t.Description(), board)
	lt, err := ParseTaskType(t.Label())
	if err != nil {
		lt = TypeTask
	}
	form.label = lt
	form.priority = t.Priority()
	form.link.SetValue(t.Link())
	form.isEdit = true
	return form
}

func (f Form) CreateTask() Task {
	id := ""
	if f.board != nil && f.board.service != nil {
		id = f.board.service.GenerateTaskID()
	}
	t := NewTaskComplete(id, f.col.status, f.title.Value(), f.description.Value(), f.priority, f.link.Value())
	t.SetLabel(f.label.String())
	return t
}

func (f Form) Init() tea.Cmd {
	return nil
}

func (f Form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case column:
		f.col = msg
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, formKeys.Quit):
			return f, tea.Quit
		case key.Matches(msg, formKeys.Back):
			if f.board != nil {
				return f.board.Update(nil)
			}
			return f, nil
		case key.Matches(msg, formKeys.Save):
			if f.board != nil {
				return f.board.Update(f)
			}
			return f, nil
		case key.Matches(msg, formKeys.Help):
			// Tab cycles through fields
			f.cycleField()
			return f, f.focusCurrentField()
		case msg.String() == "left" || msg.String() == "h":
			if f.focused == fieldLabel {
				f.label = f.prevLabel()
				return f, nil
			} else if f.focused == fieldPriority {
				f.priority = f.prevPriority()
				return f, nil
			}
		case msg.String() == "right" || msg.String() == "l":
			if f.focused == fieldLabel {
				f.label = f.nextLabel()
				return f, nil
			} else if f.focused == fieldPriority {
				f.priority = f.nextPriority()
				return f, nil
			}
		}
	}

	// Update the focused text input/area
	switch f.focused {
	case fieldTitle:
		f.title, cmd = f.title.Update(msg)
	case fieldDescription:
		f.description, cmd = f.description.Update(msg)
	case fieldLink:
		f.link, cmd = f.link.Update(msg)
	}
	return f, cmd
}

func (f *Form) cycleField() {
	// Blur current field
	switch f.focused {
	case fieldTitle:
		f.title.Blur()
	case fieldDescription:
		f.description.Blur()
	case fieldLink:
		f.link.Blur()
	}

	// Move to next field
	f.focused = (f.focused + 1) % 5
}

func (f *Form) focusCurrentField() tea.Cmd {
	switch f.focused {
	case fieldTitle:
		f.title.Focus()
		return textinput.Blink
	case fieldDescription:
		f.description.Focus()
		return textarea.Blink
	case fieldLink:
		f.link.Focus()
		return textinput.Blink
	}
	return nil
}

func (f Form) nextLabel() TaskType {
	return (f.label + 1) % TaskType(len(ValidLabels))
}

func (f Form) prevLabel() TaskType {
	if f.label == 0 {
		return TaskType(len(ValidLabels) - 1)
	}
	return f.label - 1
}

func (f Form) nextPriority() int {
	if f.priority >= 3 {
		return 1
	}
	return f.priority + 1
}

func (f Form) prevPriority() int {
	if f.priority <= 1 {
		return 3
	}
	return f.priority - 1
}

func (f Form) View() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Bold(true)

	selectedLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	formStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Margin(1, 2)

	helpStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		MarginLeft(2)

	selectorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	// Title label
	titleLabel := labelStyle
	if f.focused == fieldTitle {
		titleLabel = selectedLabelStyle
	}
	titleSection := lipgloss.JoinVertical(
		lipgloss.Left,
		titleLabel.Render("Title:"),
		f.title.View(),
	)

	// Description label
	descLabel := labelStyle
	if f.focused == fieldDescription {
		descLabel = selectedLabelStyle
	}
	descriptionSection := lipgloss.JoinVertical(
		lipgloss.Left,
		descLabel.Render("Description:"),
		f.description.View(),
	)

	// Link label
	linkLabel := labelStyle
	if f.focused == fieldLink {
		linkLabel = selectedLabelStyle
	}
	linkSection := lipgloss.JoinVertical(
		lipgloss.Left,
		linkLabel.Render("Link:"),
		f.link.View(),
	)

	// Label selector
	lblLabel := labelStyle
	if f.focused == fieldLabel {
		lblLabel = selectedLabelStyle
	}
	lblOptions := f.renderLabelOptions(selectorStyle, selectedStyle)
	lblSection := lipgloss.JoinVertical(
		lipgloss.Left,
		lblLabel.Render("Label:"),
		lblOptions,
	)

	// Priority selector
	priorityLabel := labelStyle
	if f.focused == fieldPriority {
		priorityLabel = selectedLabelStyle
	}
	priorityOptions := f.renderPriorityOptions(selectorStyle, selectedStyle)
	prioritySection := lipgloss.JoinVertical(
		lipgloss.Left,
		priorityLabel.Render("Priority:"),
		priorityOptions,
	)

	// Row for label and priority
	optionsRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lblSection,
		"    ",
		prioritySection,
	)

	header := "✨ Create a New Task"
	if f.isEdit {
		header = "✏️  Edit Task"
	}

	formContent := lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render(header),
		titleSection,
		"",
		descriptionSection,
		"",
		linkSection,
		"",
		optionsRow,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		formStyle.Render(formContent),
		helpStyle.Render(f.help.View(formKeys)),
	)
}

func (f Form) renderLabelOptions(normalStyle, selectedStyle lipgloss.Style) string {
	labels := []struct {
		t    TaskType
		name string
	}{
		{TypeTask, "task"},
		{TypeBug, "bug"},
		{TypeFeature, "feature"},
		{TypeChore, "chore"},
		{TypeDocs, "docs"},
	}

	var parts []string
	for _, opt := range labels {
		style := normalStyle
		if opt.t == f.label {
			style = selectedStyle
		}
		parts = append(parts, style.Render(opt.name))
	}

	arrows := ""
	if f.focused == fieldLabel {
		arrows = " ← → "
	} else {
		arrows = "     "
	}

	return fmt.Sprintf("[%s]%s", lipgloss.JoinHorizontal(lipgloss.Left, parts[0], " ", parts[1], " ", parts[2], " ", parts[3], " ", parts[4]), arrows)
}

func (f Form) renderPriorityOptions(normalStyle, selectedStyle lipgloss.Style) string {
	priorities := []struct {
		p    int
		name string
	}{
		{1, "High"},
		{2, "Med"},
		{3, "Low"},
	}

	var parts []string
	for _, opt := range priorities {
		style := normalStyle
		if opt.p == f.priority {
			style = selectedStyle
		}
		parts = append(parts, style.Render(opt.name))
	}

	arrows := ""
	if f.focused == fieldPriority {
		arrows = " ← → "
	} else {
		arrows = "     "
	}

	return fmt.Sprintf("[%s]%s", lipgloss.JoinHorizontal(lipgloss.Left, parts[0], " ", parts[1], " ", parts[2]), arrows)
}
