package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Show tasks ready to work on",
	Long:  `Lists tasks that have no blockers (or all blockers are done).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		tasks, err := svc.GetReadyTasks()
		if err != nil {
			output.Error(err)
		}

		var tasksJSON []task.TaskJSON
		for _, t := range tasks {
			tasksJSON = append(tasksJSON, t.ToJSON())
		}

		output.JSON(tasksJSON)
		return nil
	},
}
