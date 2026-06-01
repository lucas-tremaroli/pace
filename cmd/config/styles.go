package config

import "github.com/charmbracelet/lipgloss"

var (
	keyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)
