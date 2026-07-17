package epic

import (
	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var (
	createTitle   string
	createSummary string
	createStatus  string
)

var createCmd = &cobra.Command{
	Use:     "create",
	GroupID: "manage",
	Short:   "Create a new epic",
	Long:    `Creates a new epic and outputs the result in JSON format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithEpicService(func(svc *epic.Service) error {
			if createTitle == "" {
				return output.ErrorMsgWithCode("title is required", output.ErrCodeMissingField, "Provide a --title flag")
			}

			status, err := epic.ParseStatus(createStatus)
			if err != nil {
				return output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: planning, active, done")
			}

			e := epic.NewEpic(svc.GenerateEpicID(), status, createTitle, createSummary)
			if err := svc.CreateEpic(e); err != nil {
				return output.Error(err)
			}

			output.Success("epic created", map[string]any{"id": e.ID()})
			return nil
		})
	},
}

func init() {
	createCmd.Flags().StringVar(&createTitle, "title", "", "Epic title (required)")
	createCmd.Flags().StringVar(&createSummary, "summary", "", "Short summary of the epic")
	createCmd.Flags().StringVar(&createStatus, "status", "planning", "Epic status (planning, active, done)")
}
