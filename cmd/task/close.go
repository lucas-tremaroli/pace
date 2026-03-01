package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var closeOutcome string

var closeCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Close a task (mark as done) with an optional outcome",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		if err := svc.CloseTask(taskID, closeOutcome); err != nil {
			output.Error(err)
		}

		output.Success("task closed", map[string]any{
			"task_id": taskID,
		})
		return nil
	},
}

func init() {
	closeCmd.Flags().StringVar(&closeOutcome, "outcome", "", "Outcome message to record")
}
