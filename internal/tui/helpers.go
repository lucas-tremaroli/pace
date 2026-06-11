package tui

import (
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/muesli/reflow/truncate"
)

func truncateText(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	return truncate.StringWithTail(s, uint(maxW), "…")
}

func shortenPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func labelStyle(lbl string) lipgloss.Style {
	switch lbl {
	case "bug":
		return theme.LabelBug
	case "feature":
		return theme.LabelFeature
	case "docs":
		return theme.LabelDocs
	default:
		return theme.LabelTask
	}
}

// sortTasks: in-progress first, then todo, then done; priority asc within.
func sortTasks(tasks []task.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		order := func(s task.Status) int {
			switch s {
			case task.InProgress:
				return 0
			case task.Todo:
				return 1
			case task.Done:
				return 2
			default:
				return 3
			}
		}
		oi, oj := order(tasks[i].Status()), order(tasks[j].Status())
		if oi != oj {
			return oi < oj
		}
		return tasks[i].Priority() < tasks[j].Priority()
	})
}
