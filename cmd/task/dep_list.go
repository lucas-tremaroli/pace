package task

import (
	"errors"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var depListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List dependencies for a task",
	Long:  `Shows what tasks block the given task and what tasks it blocks.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		svc, err := task.NewService()
		if err != nil {
			return output.Error(err)
		}
		defer svc.Close()

		t, err := svc.GetTaskByID(taskID)
		if err != nil {
			if errors.Is(err, task.ErrTaskNotFound) {
				return output.ErrorMsgWithCode("task not found: "+taskID, output.ErrCodeTaskNotFound, "Use pace task list to see available task IDs")
			}
			return output.Error(err)
		}

		output.JSON(map[string]any{
			"task_id":    taskID,
			"blocked_by": t.BlockedBy(),
			"blocks":     t.Blocks(),
		})
		return nil
	},
}
