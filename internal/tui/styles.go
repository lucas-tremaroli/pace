package tui

import "github.com/charmbracelet/lipgloss"

var (
	focusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
	blurredBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("236"))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)
	listTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))
	listTitleBarStyle = lipgloss.NewStyle().MarginBottom(1)

	detailHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)
	detailTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	detailID      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	detailDesc    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	detailLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	detailValue   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	detailDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	detailLogTime = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	detailOutcome = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	detailSection = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)

	fieldBoxBorder  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	fieldBoxLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	chipBracket     = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	chipText        = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	detailMetaLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	detailLinkURL   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	statusTodo      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	statusProgress  = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	statusDone      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusBlocked   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	priorityP1      = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	priorityP2      = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	priorityP3      = lipgloss.NewStyle().Foreground(lipgloss.Color("109"))

	overviewLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	overviewPct    = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	storageTag     = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	progressFilled = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	progressEmpty  = lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	noTasksStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
)
