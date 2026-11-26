package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/epic"
)

func testRoot() (*Root, *epics) {
	ep := newEpics(&fakeEpicSvc{}, fakeTaskSvc{})
	r := &Root{
		tabs:        []tab{newKanban(nil), ep},
		help:        help.New(),
		rootKeys:    newRootKeys(),
		storagePath: "/x/proj/.pace",
		storageType: "project",
	}
	return r, ep
}

// TestViewFitsTerminal is the grid invariant: at any supported size every
// tab renders exactly the terminal's height, no line is wider than the
// terminal, and the rows are laid out as margin / tabs / body / footer.
func TestViewFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {100, 40}, {120, 30}, {200, 50}} {
		w, h := size[0], size[1]
		r, ep := testRoot()
		r.Update(tea.WindowSizeMsg{Width: w, Height: h})
		e := epic.NewEpic("epic-a1b", epic.Active, "Auth rework", "summary")
		ep.Update(epicsListedMsg{items: []list.Item{epicItem{Epic: e}}})

		for tab := range r.tabs {
			r.active = tab
			lines := strings.Split(r.View(), "\n")
			if len(lines) != h {
				t.Errorf("%dx%d tab %d: %d lines, want %d", w, h, tab, len(lines), h)
			}
			for i, l := range lines {
				if lw := lipgloss.Width(l); lw > w {
					t.Errorf("%dx%d tab %d line %d is %d wide, want <= %d", w, h, tab, i, lw, w)
				}
			}
			if strings.TrimSpace(lines[0]) != "" {
				t.Errorf("%dx%d tab %d: row 0 should be the blank top margin, got %q", w, h, tab, lines[0])
			}
			if !strings.Contains(lines[1], "Kanban") || !strings.Contains(lines[1], "Epics") {
				t.Errorf("%dx%d tab %d: row 1 should be the tab strip, got %q", w, h, tab, lines[1])
			}
			if !strings.HasPrefix(lines[2], " ╭") {
				t.Errorf("%dx%d tab %d: body should start on row 2 with a 1-column margin, got %q", w, h, tab, lines[2])
			}
			if !strings.HasPrefix(lines[h-2], " ╰") {
				t.Errorf("%dx%d tab %d: body should end on the row above the footer, got %q", w, h, tab, lines[h-2])
			}
			if strings.TrimSpace(lines[h-1]) == "" {
				t.Errorf("%dx%d tab %d: footer should be the last row", w, h, tab)
			}
		}
	}
}

func TestPaneIsExactSizeEvenWithOverflow(t *testing.T) {
	tall := strings.Repeat("a line that is much too long for the box\n", 50)
	out := pane(paneStyle(true), 30, 10, tall)
	if got := lipgloss.Height(out); got != 10 {
		t.Errorf("height = %d, want 10", got)
	}
	if got := lipgloss.Width(out); got != 30 {
		t.Errorf("width = %d, want 30", got)
	}
	iw, ih := paneInner(paneStyle(true), 30, 10)
	if iw != 26 || ih != 8 { // border 2 + padding 2 wide; border 2 tall
		t.Errorf("inner = %dx%d, want 26x8", iw, ih)
	}
}

func TestModalMatchesItsOuterSize(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {200, 50}} {
		m := newModal("hello", size[0], size[1])
		ow, oh := modalOuter(size[0], size[1])
		out := m.View()
		if lipgloss.Width(out) != ow || lipgloss.Height(out) != oh {
			t.Errorf("%v: modal is %dx%d, want %dx%d", size, lipgloss.Width(out), lipgloss.Height(out), ow, oh)
		}
	}
}
