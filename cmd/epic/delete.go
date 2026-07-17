package epic

import (
	"errors"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete [id]",
	GroupID: "manage",
	Short:   "Delete an epic",
	Long:    `Deletes an epic and unlinks any tasks that belonged to it (tasks are kept).`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithEpicService(func(svc *epic.Service) error {
			if err := svc.DeleteEpic(args[0]); err != nil {
				if errors.Is(err, epic.ErrEpicNotFound) {
					return output.ErrorMsgWithCode("epic not found: "+args[0], output.ErrCodeEpicNotFound, "Use pace epic list to see available epic IDs")
				}
				return output.Error(err)
			}
			output.Success("epic deleted", map[string]any{"id": args[0]})
			return nil
		})
	},
}
