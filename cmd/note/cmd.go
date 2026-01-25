package note

import (
	"github.com/spf13/cobra"
)

var NoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage your notes via subcommands or TUI",
	Long:  `Manage your notes via subcommands for programmatic access, or use 'pace note tui' to launch the interactive note manager.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	NoteCmd.GroupID = "core"
	NoteCmd.AddGroup(&cobra.Group{ID: "interactive", Title: "Interactive"})
	NoteCmd.AddCommand(tuiCmd)
	NoteCmd.AddCommand(listCmd)
	NoteCmd.AddCommand(createCmd)
	NoteCmd.AddCommand(readCmd)
	NoteCmd.AddCommand(deleteCmd)
}
