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
	fieldType
	fieldPriority
)

type Form struct {
	help        help.Model
	title       textinput.Model
	description textarea.Model
	link        textinput.Model
	taskType    TaskType
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
		taskType:    TypeTask,
		priority:    3,
		board:       board,
		focused:     fieldTitle,
		isEdit:      false,
	}

	form.title.Placeholder = "Task title"
	form.title.CharLimit = 50
	form.description.Placeholder = "Description (optional)"
	form.description.SetHeight(5)
	form.link.Placeholder = "Link/URL (optional)"
	form.link.CharLimit = 200
	form.title.SetValue(title)
	form.description.SetValue(description)
	form.title.Focus()
	return &form
}

// NewFormWithTask creates a form pre-populated with an existing task's values
func NewFormWithTask(t Task, board *Board) *Form {
	form := NewForm(t.Title(), t.Description(), board)
	form.taskType = t.Type()
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
	return NewTaskComplete(id, f.col.status, f.taskType, f.title.Value(), f.description.Value(), f.priority, f.link.Value())
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
			if f.focused == fieldType {
				f.taskType = f.prevType()
				return f, nil
			} else if f.focused == fieldPriority {
				f.priority = f.prevPriority()
				return f, nil
			}
		case msg.String() == "right" || msg.String() == "l":
			if f.focused == fieldType {
				f.taskType = f.nextType()
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

func (f Form) nextType() TaskType {
	return (f.taskType + 1) % 5
}

func (f Form) prevType() TaskType {
	if f.taskType == 0 {
		return 4
	}
	return f.taskType - 1
}

func (f Form) nextPriority() int {
	if f.priority >= 4 {
		return 1
	}
	return f.priority + 1
}

func (f Form) prevPriority() int {
	if f.priority <= 1 {
		return 4
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

	// Type selector
	typeLabel := labelStyle
	if f.focused == fieldType {
		typeLabel = selectedLabelStyle
	}
	typeOptions := f.renderTypeOptions(selectorStyle, selectedStyle)
	typeSection := lipgloss.JoinVertical(
		lipgloss.Left,
		typeLabel.Render("Type:"),
		typeOptions,
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

	// Row for type and priority
	optionsRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		typeSection,
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

func (f Form) renderTypeOptions(normalStyle, selectedStyle lipgloss.Style) string {
	types := []struct {
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
	for _, opt := range types {
		style := normalStyle
		if opt.t == f.taskType {
			style = selectedStyle
		}
		parts = append(parts, style.Render(opt.name))
	}

	arrows := ""
	if f.focused == fieldType {
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
		{1, "P1"},
		{2, "P2"},
		{3, "P3"},
		{4, "P4"},
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

	return fmt.Sprintf("[%s]%s", lipgloss.JoinHorizontal(lipgloss.Left, parts[0], " ", parts[1], " ", parts[2], " ", parts[3]), arrows)
}
