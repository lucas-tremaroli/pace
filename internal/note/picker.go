package note

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var listStyle = lipgloss.NewStyle().Padding(1, 2).MarginTop(1)

type noteItem struct {
	filename  string
	firstLine string
}

func (n noteItem) Title() string       { return n.filename }
func (n noteItem) Description() string { return n.firstLine }
func (n noteItem) FilterValue() string { return n.filename }

type Picker struct {
	list          list.Model
	fileToOpen    string
	shouldOpen    bool
	shouldView    bool
	quitting      bool
	service       *Service
	width         int
	height        int
	confirmForm   *huh.Form
	confirmResult *bool
	fileToDelete  string
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
			key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "view")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		}
	}
	l.AdditionalShortHelpKeys = additionalKeys
	l.AdditionalFullHelpKeys = additionalKeys

	return Picker{
		list:    l,
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
	if p.confirmForm != nil {
		form, cmd := p.confirmForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			p.confirmForm = f
			if p.confirmForm.State == huh.StateCompleted {
				if p.confirmResult != nil && *p.confirmResult {
					p.service.DeleteNote(p.fileToDelete)
					items := loadNotes(p.service.GetNotesDir())
					p.list.SetItems(items)
				}
				p.confirmForm = nil
				p.confirmResult = nil
				p.fileToDelete = ""
				return p, nil
			}
			if p.confirmForm.State == huh.StateAborted {
				p.confirmForm = nil
				p.confirmResult = nil
				p.fileToDelete = ""
				return p, nil
			}
		}
		return p, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.list.SetSize(msg.Width-4, msg.Height-4)
		return p, nil

	case tea.KeyMsg:
		return p.handleKeyMsg(msg)
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

func (p Picker) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	case "v":
		if item := p.list.SelectedItem(); item != nil {
			p.fileToOpen = item.(noteItem).filename
			p.shouldView = true
			p.quitting = true
			return p, tea.Quit
		}

	case "d":
		if item := p.list.SelectedItem(); item != nil {
			p.fileToDelete = item.(noteItem).filename
			p.confirmResult = new(bool)
			p.confirmForm = huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title(p.fileToDelete).
						Description("Delete this note?").
						Affirmative("Yes").
						Negative("No").
						Value(p.confirmResult),
				),
			)
			return p, p.confirmForm.Init()
		}
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

func (p Picker) View() string {
	if p.quitting {
		return ""
	}

	if p.confirmForm != nil {
		dialog := lipgloss.NewStyle().Width(50).Render(p.confirmForm.View())
		return lipgloss.Place(
			p.width,
			p.height,
			lipgloss.Center,
			lipgloss.Center,
			dialog,
		)
	}

	return listStyle.Render(p.list.View())
}

func (p Picker) ShouldOpenFile() bool {
	return p.shouldOpen
}

func (p Picker) ShouldViewFile() bool {
	return p.shouldView
}

func (p Picker) FileToOpen() string {
	return p.fileToOpen
}
