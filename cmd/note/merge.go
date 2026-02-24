package note

import (
	"fmt"

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
			output.ErrorMsg("output filename required (use -o flag)")
		}

		svc, err := note.NewService()
		if err != nil {
			output.Error(err)
		}

		mergedNote, err := svc.MergeNotes(args, mergeOutputFile)
		if err != nil {
			output.Error(err)
		}

		output.Success(fmt.Sprintf("Merged %d notes into %s", len(args), mergedNote.Filename), mergedNote)
		return nil
	},
}

func init() {
	mergeCmd.Flags().StringVarP(&mergeOutputFile, "output", "o", "", "Output filename for merged note (required)")
	mergeCmd.MarkFlagRequired("output")
}
