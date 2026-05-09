package note

import (
	"errors"
	"os"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <filename>",
	GroupID: "manage",
	Short:   "Delete a note",
	Long:    `Deletes a note without confirmation and outputs the result in JSON format.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]
		return cmdutil.WithNoteService(func(svc *note.Service) error {
			if err := svc.DeleteNote(filename); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					output.ErrorMsgWithCode(
						"note not found: "+filename,
						output.ErrCodeNoteNotFound,
						"Use pace note list to see available notes",
					)
				}
				output.Error(err)
			}
			output.Success("note deleted", map[string]string{"filename": filename})
			return nil
		})
	},
}
