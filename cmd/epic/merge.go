package epic

import (
	"errors"
	"fmt"

	"github.com/lucas-tremaroli/pace/v2/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/v2/internal/epic"
	"github.com/lucas-tremaroli/pace/v2/internal/output"
	"github.com/spf13/cobra"
)

var mergeTitle string

var mergeCmd = &cobra.Command{
	Use:     "merge <id1> <id2> [id3...]",
	GroupID: "manage",
	Short:   "Merge multiple epics into one",
	Long: `Merge epics into a new epic titled --title. Spec sections are combined with --- separators,
labels are combined (deduplicated), and the status is the furthest-along of the sources.
Tasks move to the new epic and the source epics are deleted.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if mergeTitle == "" {
			return output.ErrorMsgWithCode("title is required (use --title)", output.ErrCodeMissingField, "")
		}
		return cmdutil.WithEpicService(func(svc *epic.Service) error {
			merged, err := svc.MergeEpics(args, mergeTitle)
			if err != nil {
				if errors.Is(err, epic.ErrEpicNotFound) {
					return output.ErrorWithCode(err, output.ErrCodeEpicNotFound, "Use pace epic list to see available epic IDs")
				}
				return output.Error(err)
			}
			output.Success(fmt.Sprintf("merged %d epics into %s", len(args), merged.ID()), merged.ToJSON())
			return nil
		})
	},
}

func init() {
	mergeCmd.Flags().StringVar(&mergeTitle, "title", "", "Title of the merged epic (required)")
}
