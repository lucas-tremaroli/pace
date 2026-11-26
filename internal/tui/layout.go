package tui

import "github.com/charmbracelet/lipgloss"

// paneStyle is the bordered box used for every pane (kanban columns, epic
// list and detail). Padding lives here so sizing and rendering agree.
func paneStyle(focused bool) lipgloss.Style {
	if focused {
		return theme.BorderFocused.Padding(0, 1)
	}
	return theme.BorderBlurred.Padding(0, 1)
}

// pane renders content in style as a box of exactly outerW x outerH cells.
// Lip Gloss's Width/Height exclude the border and are only minimums, so the
// border is subtracted here and MaxWidth/MaxHeight cap anything that would
// otherwise overflow and shift the rows around it.
func pane(style lipgloss.Style, outerW, outerH int, content string) string {
	return style.
		Width(outerW - style.GetHorizontalBorderSize()).
		Height(outerH - style.GetVerticalBorderSize()).
		MaxWidth(outerW).
		MaxHeight(outerH).
		Render(content)
}

// paneInner returns the content area of a pane with the given outer size:
// what remains after the style's border and padding.
func paneInner(style lipgloss.Style, outerW, outerH int) (w, h int) {
	return outerW - style.GetHorizontalFrameSize(), outerH - style.GetVerticalFrameSize()
}
