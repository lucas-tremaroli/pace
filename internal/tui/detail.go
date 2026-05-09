package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/muesli/reflow/wrap"
)

// LogEntry holds a single task log entry for rendering.
type LogEntry struct {
	Time      string
	IsOutcome bool
	Message   string
}

// renderFieldBox builds a bordered box for a single field.
func renderFieldBox(label, value string, valueStyle lipgloss.Style, boxWidth int) string {
	innerW := boxWidth - 2

	topLabel := " " + label + " "
	fillLen := innerW - len(topLabel)
	if fillLen < 0 {
		fillLen = 0
	}
	top := fieldBoxBorder.Render("┌") + fieldBoxLabel.Render(topLabel) + fieldBoxBorder.Render(strings.Repeat("─", fillLen)+"┐")

	styledVal := valueStyle.Render(value)
	valVisualW := lipgloss.Width(styledVal)
	padLen := innerW - 1 - valVisualW
	if padLen < 0 {
		padLen = 0
	}
	mid := fieldBoxBorder.Render("│") + " " + styledVal + strings.Repeat(" ", padLen) + fieldBoxBorder.Render("│")

	bot := fieldBoxBorder.Render("└" + strings.Repeat("─", innerW) + "┘")

	return top + "\n" + mid + "\n" + bot
}

// renderFieldBoxes renders Status, Priority, and Label boxes side-by-side.
func renderFieldBoxes(tk task.Task, w int) string {
	const boxW = 14
	const gap = 2

	var statusText string
	var statusStyle lipgloss.Style
	if tk.IsBlocked() {
		statusText = "⊘ blocked"
		statusStyle = statusBlocked
	} else {
		switch tk.Status() {
		case task.Todo:
			statusText = "○ todo"
			statusStyle = statusTodo
		case task.InProgress:
			statusText = "● active"
			statusStyle = statusProgress
		case task.Done:
			statusText = "● done"
			statusStyle = statusDone
		default:
			statusText = tk.Status().String()
			statusStyle = detailValue
		}
	}

	var priText string
	var priStyle lipgloss.Style
	switch tk.Priority() {
	case 1:
		priText = "! high"
		priStyle = priorityP1
	case 2:
		priText = "~ medium"
		priStyle = priorityP2
	case 3:
		priText = "· low"
		priStyle = priorityP3
	default:
		priText = task.PriorityName(tk.Priority())
		priStyle = detailValue
	}

	lbl := tk.Label()
	if lbl == "" {
		lbl = "task"
	}

	box1 := renderFieldBox("Status", statusText, statusStyle, boxW)
	box2 := renderFieldBox("Label", lbl, labelStyle(lbl), boxW)
	box3 := renderFieldBox("Priority", priText, priStyle, boxW)

	if w < 46 {
		return box1 + "\n" + box2 + "\n" + box3
	}

	spacer := strings.Repeat(" ", gap)
	return lipgloss.JoinHorizontal(lipgloss.Top, box1, spacer, box2, spacer, box3)
}

// renderChips renders items as [item1] [item2] with styled brackets.
func renderChips(items []string) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = chipBracket.Render("[") + chipText.Render(item) + chipBracket.Render("]")
	}
	return strings.Join(parts, " ")
}

func renderTaskDetail(tk task.Task, w int, logs []LogEntry) string {
	wrapStyled := func(s string, style lipgloss.Style) string {
		lines := strings.Split(wrap.String(s, w), "\n")
		for i, line := range lines {
			lines[i] = style.Render(line)
		}
		return strings.Join(lines, "\n")
	}

	wrapIndent := func(s string, indent int, style lipgloss.Style) string {
		iw := w - indent
		if iw < 10 {
			iw = 10
		}
		lines := strings.Split(wrap.String(s, iw), "\n")
		pad := strings.Repeat(" ", indent)
		for i, line := range lines {
			if i > 0 {
				lines[i] = pad + style.Render(line)
			} else {
				lines[i] = style.Render(line)
			}
		}
		return strings.Join(lines, "\n")
	}

	var b strings.Builder

	b.WriteString(wrapStyled(tk.Title(), detailTitle))
	b.WriteString("\n")
	b.WriteString(detailID.Render(tk.ID()))
	b.WriteString("\n\n")
	b.WriteString(renderFieldBoxes(tk, w))
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(detailSection.Render("Description"))
	b.WriteString("\n\n")
	if desc := tk.Description(); desc != "" {
		b.WriteString(wrapStyled(desc, detailDesc))
	} else {
		b.WriteString(detailDim.Render("No description"))
	}
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(detailSection.Render("Metadata"))
	b.WriteString("\n\n")

	hasLink := tk.Link() != ""
	hasBlockedBy := len(tk.BlockedBy()) > 0
	hasBlocks := len(tk.Blocks()) > 0
	hasNotes := len(tk.Notes()) > 0
	if hasLink || hasBlockedBy || hasBlocks || hasNotes {
		const metaLabelW = 13
		metaRow := func(label, value string) {
			padded := label + ":" + strings.Repeat(" ", metaLabelW-len(label)-1)
			b.WriteString(detailMetaLabel.Render(padded))
			b.WriteString(value)
			b.WriteString("\n")
		}

		if hasLink {
			metaRow("Link", wrapIndent(tk.Link(), metaLabelW, detailLinkURL))
		}
		if hasBlockedBy {
			metaRow("Blocked by", renderChips(tk.BlockedBy()))
		}
		if hasBlocks {
			metaRow("Blocks", renderChips(tk.Blocks()))
		}
		if hasNotes {
			metaRow("Notes", renderChips(tk.Notes()))
		}
	} else {
		b.WriteString(detailDim.Render("No metadata"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(detailLabel.Render(fmt.Sprintf("Logs (%d):", len(logs))))
	b.WriteString("\n")
	if len(logs) == 0 {
		b.WriteString(detailDim.Render("  No logs yet"))
		b.WriteString("\n")
	} else {
		for _, entry := range logs {
			styledPrefix := detailLogTime.Render(entry.Time + " ")
			prefixW := len(entry.Time) + 1
			if entry.IsOutcome {
				styledPrefix += detailOutcome.Render("[outcome] ")
				prefixW += len("[outcome] ")
			}
			indent := 2 + prefixW
			b.WriteString("  ")
			b.WriteString(styledPrefix)
			b.WriteString(wrapIndent(entry.Message, indent, detailValue))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func renderNoteDetail(n note.Note, w int) string {
	var b strings.Builder
	b.WriteString(detailHeader.Render(wrap.String(n.Filename, w)))
	b.WriteString("\n")

	if n.Description != "" {
		b.WriteString(detailLabel.Render(wrap.String(n.Description, w)))
		b.WriteString("\n\n")
	}

	if len(n.Tasks) > 0 {
		b.WriteString(detailLabel.Render("Linked tasks: "))
		b.WriteString(detailValue.Render(wrap.String(strings.Join(n.Tasks, ", "), w)))
		b.WriteString("\n\n")
	}

	if n.Content != "" {
		b.WriteString(note.RenderMarkdownWithWidth(n.Content, w))
	}

	return b.String()
}
