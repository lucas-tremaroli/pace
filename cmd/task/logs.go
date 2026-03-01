package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "View log entries for a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		logs, err := svc.GetTaskLogs(taskID)
		if err != nil {
			output.Error(err)
		}

		output.JSON(map[string]any{
			"task_id": taskID,
			"logs":    logs,
			"count":   len(logs),
		})
		return nil
	},
}
