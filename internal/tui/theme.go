package tui

import "github.com/charmbracelet/lipgloss"

// Theme is a self-contained copy of the styles used by the prototype TUI.
// Decoupled from internal/tui so the two TUIs can evolve independently.
type Theme struct {
	// Borders & chrome
	BorderFocused lipgloss.Style
	BorderBlurred lipgloss.Style
	Help          lipgloss.Style
	ListTitle     lipgloss.Style
	ListTitleBar  lipgloss.Style

	// Tab strip
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style

	// Detail panel
	DetailHeader    lipgloss.Style
	DetailTitle     lipgloss.Style
	DetailID        lipgloss.Style
	DetailDesc      lipgloss.Style
	DetailLabel     lipgloss.Style
	DetailValue     lipgloss.Style
	DetailDim       lipgloss.Style
	DetailLogTime   lipgloss.Style
	DetailOutcome   lipgloss.Style
	DetailSection   lipgloss.Style
	DetailMetaLabel lipgloss.Style
	DetailLinkURL   lipgloss.Style
	FieldBoxBorder  lipgloss.Style
	FieldBoxLabel   lipgloss.Style
	ChipBracket     lipgloss.Style
	ChipText        lipgloss.Style

	// Status & priority (detail-panel variant)
	StatusTodo       lipgloss.Style
	StatusInProgress lipgloss.Style
	StatusDone       lipgloss.Style
	StatusBlocked    lipgloss.Style
	PriorityHigh     lipgloss.Style
	PriorityMedium   lipgloss.Style
	PriorityLow      lipgloss.Style

	// Overview / header
	OverviewLabel  lipgloss.Style
	OverviewPct    lipgloss.Style
	StorageTag     lipgloss.Style
	ProgressFilled lipgloss.Style
	ProgressEmpty  lipgloss.Style
	NoTasks        lipgloss.Style

	// Task list item
	TaskNormal         lipgloss.Style
	TaskInProgressIcon lipgloss.Style
	TaskDoneIcon       lipgloss.Style
	TaskDoneText       lipgloss.Style
	TaskBlockedIcon    lipgloss.Style
	TaskBlockedText    lipgloss.Style
	TaskSelected       lipgloss.Style
	TaskMeta           lipgloss.Style
	ItemPriorityHigh   lipgloss.Style
	ItemPriorityMedium lipgloss.Style
	ItemPriorityLow    lipgloss.Style

	// Labels
	LabelTask    lipgloss.Style
	LabelBug     lipgloss.Style
	LabelFeature lipgloss.Style
	LabelDocs    lipgloss.Style
}

func DefaultTheme() Theme {
	return Theme{
		BorderFocused: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")),
		BorderBlurred: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("236")),
		Help:          lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
		ListTitle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")),
		ListTitleBar:  lipgloss.NewStyle().MarginBottom(1),

		TabActive:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		TabInactive: lipgloss.NewStyle().Foreground(lipgloss.Color("241")),

		DetailHeader:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1),
		DetailTitle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")),
		DetailID:         lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		DetailDesc:       lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		DetailLabel:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true),
		DetailValue:      lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		DetailDim:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		DetailLogTime:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		DetailOutcome:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		DetailSection:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true),
		DetailMetaLabel:  lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true),
		DetailLinkURL:    lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		FieldBoxBorder:   lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		FieldBoxLabel:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true),
		ChipBracket:      lipgloss.NewStyle().Foreground(lipgloss.Color("62")),
		ChipText:         lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		StatusTodo:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		StatusInProgress: lipgloss.NewStyle().Foreground(lipgloss.Color("226")),
		StatusDone:       lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		StatusBlocked:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		PriorityHigh:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		PriorityMedium:   lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true),
		PriorityLow:      lipgloss.NewStyle().Foreground(lipgloss.Color("109")),
		OverviewLabel:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		OverviewPct:      lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true),
		StorageTag:       lipgloss.NewStyle().Foreground(lipgloss.Color("62")),
		ProgressFilled:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		ProgressEmpty:    lipgloss.NewStyle().Foreground(lipgloss.Color("236")),
		NoTasks:          lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true),

		TaskNormal:         lipgloss.NewStyle(),
		TaskInProgressIcon: lipgloss.NewStyle().Foreground(lipgloss.Color("226")),
		TaskDoneIcon:       lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		TaskDoneText:       lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Strikethrough(true),
		TaskBlockedIcon:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		TaskBlockedText:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		TaskSelected:       lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true),
		TaskMeta:           lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		ItemPriorityHigh:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		ItemPriorityMedium: lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true),
		ItemPriorityLow:    lipgloss.NewStyle().Foreground(lipgloss.Color("146")),

		LabelTask:    lipgloss.NewStyle().Foreground(lipgloss.Color("75")),
		LabelBug:     lipgloss.NewStyle().Foreground(lipgloss.Color("167")),
		LabelFeature: lipgloss.NewStyle().Foreground(lipgloss.Color("114")),
		LabelDocs:    lipgloss.NewStyle().Foreground(lipgloss.Color("141")),
	}
}

var theme = DefaultTheme()
