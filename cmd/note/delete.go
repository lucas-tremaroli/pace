package note

import (
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <filename>",
	Short: "Delete a note",
	Long:  `Deletes a note without confirmation and outputs the result in JSON format.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		svc, err := note.NewService()
		if err != nil {
			output.Error(err)
		}

		if err := svc.DeleteNote(filename); err != nil {
			output.Error(err)
		}

		output.Success("note deleted", map[string]string{
			"filename": filename,
		})
		return nil
	},
}
