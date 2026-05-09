package task

import (
	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:     "logs <id>",
	GroupID: "logging",
	Short:   "View log entries for a task",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		return cmdutil.WithTaskService(func(svc *task.Service) error {
			logs, err := svc.GetTaskLogs(taskID)
			if err != nil {
				return output.Error(err)
			}
			output.JSON(map[string]any{
				"task_id": taskID,
				"logs":    logs,
				"count":   len(logs),
			})
			return nil
		})
	},
}
