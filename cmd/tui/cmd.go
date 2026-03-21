package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/tui"
	"github.com/spf13/cobra"
)

var TuiCmd = &cobra.Command{
	Use:     "tui",
	GroupID: "core",
	Short:   "Open the unified TUI dashboard",
	Long:    `Launch an interactive TUI showing tasks and notes in a two-panel layout.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := tui.NewTui()
		if err != nil {
			return fmt.Errorf("failed to initialize TUI: %w", err)
		}
		p := tea.NewProgram(t, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}
		return nil
	},
}
