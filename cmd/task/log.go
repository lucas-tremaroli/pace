package task

import (
	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:     "log <id> <message>",
	GroupID: "logging",
	Short:   "Add a progress log entry to a task",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID, message := args[0], args[1]
		return cmdutil.WithTaskService(func(svc *task.Service) error {
			if err := svc.LogEntry(taskID, message); err != nil {
				output.Error(err)
			}
			output.Success("log entry added", map[string]any{"task_id": taskID})
			return nil
		})
	},
}
