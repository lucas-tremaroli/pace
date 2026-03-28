package note

import (
	"github.com/spf13/cobra"
)

var NoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage your notes via subcommands",
	Long:  `Manage your notes via subcommands for programmatic access, or use 'pace tui' to launch the interactive dashboard.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	NoteCmd.GroupID = "core"
	NoteCmd.AddGroup(&cobra.Group{ID: "manage", Title: "Manage"})
	NoteCmd.AddCommand(listCmd)
	NoteCmd.AddCommand(createCmd)
	NoteCmd.AddCommand(readCmd)
	NoteCmd.AddCommand(deleteCmd)
	NoteCmd.AddCommand(mergeCmd)
}
