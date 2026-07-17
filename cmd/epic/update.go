package epic

import (
	"errors"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var (
	updateTitle   string
	updateSummary string
	updateStatus  string
)

var updateCmd = &cobra.Command{
	Use:     "update [id]",
	GroupID: "manage",
	Short:   "Update an existing epic",
	Long:    `Updates an epic. Only specified fields are changed.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithEpicService(func(svc *epic.Service) error {
			e, err := svc.GetEpicByID(args[0])
			if err != nil {
				if errors.Is(err, epic.ErrEpicNotFound) {
					return output.ErrorMsgWithCode("epic not found: "+args[0], output.ErrCodeEpicNotFound, "Use pace epic list to see available epic IDs")
				}
				return output.Error(err)
			}

			if cmd.Flags().Changed("title") {
				e.SetTitle(updateTitle)
			}
			if cmd.Flags().Changed("summary") {
				e.SetSummary(updateSummary)
			}
			if cmd.Flags().Changed("status") {
				status, err := epic.ParseStatus(updateStatus)
				if err != nil {
					return output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: planning, active, done")
				}
				if err := e.SetStatus(status); err != nil {
					return output.Error(err)
				}
			}

			if err := svc.UpdateEpic(*e); err != nil {
				return output.Error(err)
			}
			output.Success("epic updated", e.ToJSON())
			return nil
		})
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateTitle, "title", "", "Epic title")
	updateCmd.Flags().StringVar(&updateSummary, "summary", "", "Epic summary")
	updateCmd.Flags().StringVar(&updateStatus, "status", "", "Epic status (planning, active, done)")
}
