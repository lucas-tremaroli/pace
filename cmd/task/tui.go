package task

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:     "tui",
	GroupID: "interactive",
	Short:   "Launch the Kanban board TUI",
	Long:    `Launch an interactive TUI to manage your tasks in a Kanban-style board.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		board, err := task.NewBoard()
		if err != nil {
			return fmt.Errorf("failed to initialize task board: %w", err)
		}
		p := tea.NewProgram(board, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running program: %w", err)
		}
		return nil
	},
}
