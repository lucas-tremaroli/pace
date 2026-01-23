package task

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var TaskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage your tasks in a TUI or via subcommands",
	Long:  `Launch a TUI to manage your tasks, or use subcommands for programmatic access.`,
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

func init() {
	TaskCmd.AddCommand(listCmd)
	TaskCmd.AddCommand(getCmd)
	TaskCmd.AddCommand(createCmd)
	TaskCmd.AddCommand(updateCmd)
	TaskCmd.AddCommand(deleteCmd)
}
