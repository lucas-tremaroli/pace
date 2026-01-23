package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

type taskListResponse struct {
	Tasks []task.TaskJSON `json:"tasks"`
	Count int             `json:"count"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks as JSON",
	Long:  `Outputs all tasks in JSON format for programmatic access.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		tasks, err := svc.LoadAllTasks()
		if err != nil {
			output.Error(err)
		}

		taskJSONs := make([]task.TaskJSON, len(tasks))
		for i, t := range tasks {
			taskJSONs[i] = t.ToJSON()
		}

		output.JSON(taskListResponse{
			Tasks: taskJSONs,
			Count: len(taskJSONs),
		})
		return nil
	},
}
