package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task by ID",
	Long:  `Deletes a task without confirmation and outputs the result in JSON format.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		if err := svc.DeleteTask(taskID); err != nil {
			output.Error(err)
		}

		output.Success("task deleted", map[string]string{
			"id": taskID,
		})
		return nil
	},
}
