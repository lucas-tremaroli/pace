package note

import (
	"github.com/spf13/cobra"
)

var NoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage your markdown notes",
	Long:  `Create, list, and manage your markdown notes with ease.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	NoteCmd.AddCommand(listCmd)
	NoteCmd.AddCommand(createCmd)
	NoteCmd.AddCommand(readCmd)
	NoteCmd.AddCommand(deleteCmd)
}
