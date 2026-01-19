package note

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	listStyle = lipgloss.NewStyle().Padding(1, 2).MarginTop(1)

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			MarginLeft(2).
			MarginTop(1)

	messageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			MarginLeft(2)
)

type pickerState int

const (
	stateBrowsing pickerState = iota
	stateConfirmDelete
)

type noteItem struct {
	filename  string
	firstLine string
}

func (n noteItem) Title() string       { return n.filename }
func (n noteItem) Description() string { return n.firstLine }
func (n noteItem) FilterValue() string { return n.filename }

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
	delegate.ShowDescription = true

	l := list.New(items, delegate, 0, 0)
	l.Title = "Notes"
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	additionalKeys := func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("o", "enter"), key.WithHelp("o", "open")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		}
	}
	l.AdditionalShortHelpKeys = additionalKeys
	l.AdditionalFullHelpKeys = additionalKeys

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
			firstLine := readFirstLine(filepath.Join(dir, e.Name()))
			items = append(items, noteItem{filename: e.Name(), firstLine: firstLine})
		}
	}
	return items
}

func readFirstLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Strip leading markdown heading markers
		line = strings.TrimLeft(line, "# ")
		return line
	}
	return ""
}

func (p Picker) Init() tea.Cmd {
	return nil
}

func (p Picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		// Account for padding
		p.list.SetSize(msg.Width-4, msg.Height-4)
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
	case "ctrl+c":
		p.quitting = true
		return p, tea.Quit

	case "q", "esc":
		// Don't quit if filtering - let the list handle it
		if p.list.FilterState() == list.Filtering {
			break
		}
		p.quitting = true
		return p, tea.Quit

	case "enter", "o":
		if item := p.list.SelectedItem(); item != nil {
			p.fileToOpen = item.(noteItem).filename
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
			filename := item.(noteItem).filename
			if err := p.service.DeleteNote(filename); err != nil {
				p.message = "Error: " + err.Error()
			} else {
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

	listView := listStyle.Render(p.list.View())

	if p.state == stateConfirmDelete {
		if item := p.list.SelectedItem(); item != nil {
			status := confirmStyle.Render("Delete " + item.(noteItem).filename + "? [y/n]")
			return lipgloss.JoinVertical(lipgloss.Left, listView, status)
		}
	} else if p.message != "" {
		status := messageStyle.Render(p.message)
		return lipgloss.JoinVertical(lipgloss.Left, listView, status)
	}

	return listView
}

func (p Picker) ShouldOpenFile() bool {
	return p.shouldOpen
}

func (p Picker) FileToOpen() string {
	return p.fileToOpen
}
