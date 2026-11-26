package epic

import (
	"errors"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:     "edit [id]",
	GroupID: "manage",
	Short:   "Edit an epic in $EDITOR",
	Long: `Opens the epic as a markdown document in $EDITOR (falling back to nvim, vim, vi, nano).
The frontmatter holds title, summary, status and labels; each "## " section is a spec section.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithEpicService(func(svc *epic.Service) error {
			e, err := svc.GetEpicByID(args[0])
			if err != nil {
				if errors.Is(err, epic.ErrEpicNotFound) {
					return output.ErrorMsgWithCode("epic not found: "+args[0], output.ErrCodeEpicNotFound, "Use pace epic list to see available epic IDs")
				}
				return output.Error(err)
			}

			edited, err := cmdutil.EditText(epic.Format(*e))
			if err != nil {
				return output.Error(err)
			}
			updated, err := epic.Parse(edited, *e)
			if err != nil {
				return output.ErrorWithCode(err, output.ErrCodeInvalidParams, rejectedHint(edited))
			}
			if err := svc.UpdateEpic(updated); err != nil {
				return output.Error(err)
			}
			output.Success("epic updated", updated.ToJSON())
			return nil
		})
	},
}

// rejectedHint tells the user where their rejected edit was saved.
func rejectedHint(edited string) string {
	if path := cmdutil.KeepText(edited); path != "" {
		return "Your edit was kept in " + path
	}
	return "Fix the document and try again"
}
