package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id> [id2] [id3] ...",
	Short: "Delete one or more tasks by ID",
	Long:  `Deletes one or more tasks without confirmation and outputs the result in JSON format.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		// Single ID: backward compatible behavior
		if len(args) == 1 {
			taskID := args[0]
			if err := svc.DeleteTask(taskID); err != nil {
				output.Error(err)
			}
			output.Success("task deleted", map[string]string{
				"id": taskID,
			})
			return nil
		}

		// Multiple IDs: bulk delete
		result := output.BulkResult{
			Total: len(args),
		}

		for _, taskID := range args {
			if err := svc.DeleteTask(taskID); err != nil {
				result.Failed = append(result.Failed, output.BulkItem{
					ID:    taskID,
					Error: err.Error(),
				})
			} else {
				result.Succeeded = append(result.Succeeded, output.BulkItem{
					ID: taskID,
				})
			}
		}

		output.BulkSuccess("tasks deleted", result)
		return nil
	},
}
