package note

import (
	"errors"
	"fmt"
	"os"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var mergeOutputFile string

var mergeCmd = &cobra.Command{
	Use:   "merge <note1> <note2> [note3...]",
	Short: "Merge multiple notes into one",
	Long: `Merge multiple notes into a single output note.
Labels from all source notes are combined (deduplicated).
Content is concatenated with --- separators.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if mergeOutputFile == "" {
			output.ErrorMsgWithCode("output filename required (use -o flag)", output.ErrCodeMissingField, "")
		}
		return cmdutil.WithNoteService(func(svc *note.Service) error {
			mergedNote, err := svc.MergeNotes(args, mergeOutputFile)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					output.ErrorWithCode(err, output.ErrCodeNoteNotFound, "Use pace note list to see available notes")
				}
				output.ErrorWithCode(err, output.ErrCodeStorageError, "")
			}
			output.Success(fmt.Sprintf("Merged %d notes into %s", len(args), mergedNote.Filename), mergedNote)
			return nil
		})
	},
}

func init() {
	mergeCmd.Flags().StringVarP(&mergeOutputFile, "output", "o", "", "Output filename for merged note (required)")
	mergeCmd.MarkFlagRequired("output")
}
