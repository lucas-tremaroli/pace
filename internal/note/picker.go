package note

import (
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	listStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Margin(1, 2)

	helpBoxStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			MarginLeft(2)

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			MarginLeft(5).
			MarginTop(1)

	messageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			MarginLeft(5).
			MarginTop(1)

	helpTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type pickerState int

const (
	stateBrowsing pickerState = iota
	stateConfirmDelete
)

type noteItem string

func (n noteItem) Title() string       { return string(n) }
func (n noteItem) Description() string { return "" }
func (n noteItem) FilterValue() string { return string(n) }

type Picker struct {
	list       list.Model
	state      pickerState
	fileToOpen string
	shouldOpen bool
	quitting   bool
	service    *Service
	message    string
	width      int
	height     int
}

func NewPicker(svc *Service) Picker {
	items := loadNotes(svc.GetNotesDir())

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	l := list.New(items, delegate, 0, 0)
	l.Title = "Notes"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).MarginLeft(1)

	return Picker{
		list:    l,
		state:   stateBrowsing,
		service: svc,
	}
}

func loadNotes(dir string) []list.Item {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var items []list.Item
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > 3 && e.Name()[len(e.Name())-3:] == ".md" {
			items = append(items, noteItem(e.Name()))
		}
	}
	return items
}

func (p Picker) Init() tea.Cmd {
	return nil
}

func (p Picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		// Account for borders, padding, margins
		listWidth := msg.Width - 12
		listHeight := msg.Height - 22
		p.list.SetSize(listWidth, listHeight)
		return p, nil

	case tea.KeyMsg:
		if p.state == stateConfirmDelete {
			return p.updateConfirmDelete(msg)
		}
		return p.updateBrowsing(msg)
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

func (p Picker) updateBrowsing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		p.quitting = true
		return p, tea.Quit

	case "enter", "o":
		if item := p.list.SelectedItem(); item != nil {
			p.fileToOpen = string(item.(noteItem))
			p.shouldOpen = true
			p.quitting = true
			return p, tea.Quit
		}

	case "d":
		if p.list.SelectedItem() != nil {
			p.state = stateConfirmDelete
			return p, nil
		}
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

func (p Picker) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if item := p.list.SelectedItem(); item != nil {
			filename := string(item.(noteItem))
			if err := p.service.DeleteNote(filename); err != nil {
				p.message = "Error: " + err.Error()
			} else {
				p.message = "Deleted " + filename
				// Refresh list
				items := loadNotes(p.service.GetNotesDir())
				p.list.SetItems(items)
			}
		}
		p.state = stateBrowsing
		return p, nil

	case "n", "esc":
		p.state = stateBrowsing
		return p, nil

	case "q", "ctrl+c":
		p.quitting = true
		return p, tea.Quit
	}

	return p, nil
}

func (p Picker) View() string {
	if p.quitting {
		return ""
	}

	// List in bordered box with margin
	listView := listStyle.Render(p.list.View())

	// Help box
	var helpText string
	if p.state == stateConfirmDelete {
		helpText = "y confirm • n/esc cancel • q quit"
	} else {
		helpText = "↑/k up • ↓/j down • enter/o open • d delete • esc/q quit"
	}
	helpView := helpBoxStyle.Render(helpTextStyle.Render(helpText))

	// Add status message if present
	var statusView string
	if p.state == stateConfirmDelete {
		if item := p.list.SelectedItem(); item != nil {
			statusView = confirmStyle.Render("Delete " + string(item.(noteItem)) + "? [y/n]")
		}
	} else if p.message != "" {
		statusView = messageStyle.Render(p.message)
	}

	// Combine list, help, and status
	if statusView != "" {
		return lipgloss.JoinVertical(lipgloss.Left, listView, helpView, statusView)
	}
	return lipgloss.JoinVertical(lipgloss.Left, listView, helpView)
}

func (p Picker) ShouldOpenFile() bool {
	return p.shouldOpen
}

func (p Picker) FileToOpen() string {
	return p.fileToOpen
}
