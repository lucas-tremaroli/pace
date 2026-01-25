package task

import (
	"github.com/spf13/cobra"
)

var TaskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage your tasks via subcommands or TUI",
	Long:  `Manage your tasks via subcommands for programmatic access, or use 'pace task tui' to launch the Kanban board.`,
}

func init() {
	TaskCmd.GroupID = "core"
	TaskCmd.AddGroup(&cobra.Group{ID: "interactive", Title: "Interactive"})
	TaskCmd.AddCommand(tuiCmd)
	TaskCmd.AddCommand(listCmd)
	TaskCmd.AddCommand(getCmd)
	TaskCmd.AddCommand(createCmd)
	TaskCmd.AddCommand(updateCmd)
	TaskCmd.AddCommand(deleteCmd)
	TaskCmd.AddCommand(depCmd)
	TaskCmd.AddCommand(readyCmd)
	TaskCmd.AddCommand(searchCmd)
}
