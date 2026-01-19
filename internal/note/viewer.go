package note

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	viewerStyle = lipgloss.NewStyle().Padding(1, 2)
)

var keys = struct {
	Quit     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	HalfUp   key.Binding
	HalfDown key.Binding
}{
	Quit:     key.NewBinding(key.WithKeys("q", "esc")),
	Top:      key.NewBinding(key.WithKeys("g", "home")),
	Bottom:   key.NewBinding(key.WithKeys("G", "end")),
	HalfUp:   key.NewBinding(key.WithKeys("u", "ctrl+u")),
	HalfDown: key.NewBinding(key.WithKeys("d", "ctrl+d")),
}

type Viewer struct {
	viewport        viewport.Model
	filename        string
	renderedContent string
	ready           bool
}

func NewViewer(filename, renderedContent string) Viewer {
	return Viewer{
		filename:        filename,
		renderedContent: renderedContent,
	}
}

func RenderMarkdown(content string) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}

	return rendered
}

func (v Viewer) Init() tea.Cmd {
	return nil
}

func (v Viewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 3
		footerHeight := 1
		verticalMarginHeight := headerHeight + footerHeight

		if !v.ready {
			v.viewport = viewport.New(msg.Width-4, msg.Height-verticalMarginHeight)
			v.viewport.YPosition = headerHeight
			v.viewport.SetContent(v.renderedContent)
			v.ready = true
		} else {
			v.viewport.Width = msg.Width - 4
			v.viewport.Height = msg.Height - verticalMarginHeight
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return v, tea.Quit
		case key.Matches(msg, keys.Top):
			v.viewport.GotoTop()
		case key.Matches(msg, keys.Bottom):
			v.viewport.GotoBottom()
		case key.Matches(msg, keys.HalfUp):
			v.viewport.HalfPageUp()
		case key.Matches(msg, keys.HalfDown):
			v.viewport.HalfPageDown()
		}
	}

	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

func (v Viewer) View() string {
	if !v.ready {
		return "Loading..."
	}

	header := titleStyle.Render(v.filename)
	footer := v.footerView()

	return viewerStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			v.viewport.View(),
			footer,
		),
	)
}

func (v Viewer) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", v.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, v.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}
