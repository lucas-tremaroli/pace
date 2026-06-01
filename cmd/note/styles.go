package note

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	headingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
)
