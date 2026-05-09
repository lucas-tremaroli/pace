package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/task"
)

func (t *Tui) listWidth() int {
	w := int(float64(t.width) * 0.35)
	if w < minListW {
		w = minListW
	}
	if w > t.width-minListW {
		w = t.width - minListW
	}
	return w
}

func (t *Tui) detailWidth() int {
	return t.width - t.listWidth()
}

// contentWidth returns the usable character width inside the detail viewport,
// accounting for the border (2) and inner padding (2).
func (t *Tui) contentWidth() int {
	w := t.detailWidth() - 4
	if w < 20 {
		w = 20
	}
	return w
}

// fitHeight ensures s is exactly h lines tall, truncating or padding as needed.
func fitHeight(s string, h int) string {
	if h <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (t *Tui) recalcLayout() {
	helpH := t.cachedHelpH
	if t.width != t.cachedHelpWidth {
		t.help.Width = t.width - 2
		helpH = lipgloss.Height(t.help.View(tuiKeys))
		t.cachedHelpH = helpH
		t.cachedHelpWidth = t.width
	}

	availH := t.height - helpH
	lw := t.listWidth() - 2
	dw := t.detailWidth() - 2

	listH := availH - overviewH
	taskH := listH / 2
	noteH := listH - taskH

	t.taskList.SetSize(lw, taskH-2)
	t.noteList.SetSize(lw, noteH-2)
	t.viewport.Width = dw
	t.viewport.Height = availH - 2

	t.layoutAvailH = availH
	t.layoutTaskH = taskH
	t.layoutNoteH = noteH
}

func shortenPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func (t *Tui) renderOverview(w int) string {
	var total, todo, inProg, done int
	for _, item := range t.taskList.Items() {
		if ti, ok := item.(TaskItem); ok {
			total++
			switch ti.Task.Status() {
			case task.Todo:
				todo++
			case task.InProgress:
				inProg++
			case task.Done:
				done++
			}
		}
	}

	tag := storageTag.Render(strings.ToUpper(string(t.storageType)))
	storeLine := overviewLabel.Render(shortenPath(t.storagePath)) + "  " + tag

	if total == 0 {
		content := "\n" + storeLine + "\n\n" + noTasksStyle.Render("no tasks yet")
		return fitHeight(content, overviewH)
	}

	pct := done * 100 / total
	pctStr := overviewPct.Render(fmt.Sprintf(" %d%%", pct))
	barW := w - 6
	if barW < 10 {
		barW = 10
	}
	filled := done * barW / total
	bar := progressFilled.Render(strings.Repeat("━", filled)) +
		progressEmpty.Render(strings.Repeat("─", barW-filled)) +
		pctStr

	counts := overviewLabel.Render(fmt.Sprintf("Progress %d/%d  ", done, total)) +
		statusTodo.Render(fmt.Sprintf("○ %d", todo)) +
		overviewLabel.Render("  ") +
		statusProgress.Render(fmt.Sprintf("● %d", inProg)) +
		overviewLabel.Render("  ") +
		statusDone.Render(fmt.Sprintf("✓ %d", done))

	content := "\n" + storeLine + "\n\n" + bar + "\n\n" + counts
	return fitHeight(content, overviewH)
}
