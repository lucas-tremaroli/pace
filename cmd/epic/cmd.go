package epic

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// EpicCmd is the root command for managing epics.
var EpicCmd = &cobra.Command{
	Use:   "epic",
	Short: "Manage epics via subcommands",
	Long:  `Manage epics — spec-first containers that group tasks. Use subcommands for programmatic access.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var (
	headingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	idStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

func init() {
	EpicCmd.GroupID = "core"
	EpicCmd.AddGroup(&cobra.Group{ID: "manage", Title: "Manage"})
	EpicCmd.AddCommand(createCmd)
	EpicCmd.AddCommand(listCmd)
	EpicCmd.AddCommand(getCmd)
	EpicCmd.AddCommand(updateCmd)
	EpicCmd.AddCommand(deleteCmd)
	EpicCmd.AddCommand(specCmd)
}
