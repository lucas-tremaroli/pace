package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	createTitle       string
	createDescription string
	createStatus      string
	createPriority    int
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	Long:  `Creates a new task and outputs the result in JSON format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if createTitle == "" {
			output.ErrorMsg("title is required")
		}

		status, err := task.ParseStatus(createStatus)
		if err != nil {
			output.Error(err)
		}

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		newTask := task.NewTaskWithPriority(svc.GenerateTaskID(), status, createTitle, createDescription, createPriority)

		if err := svc.CreateTask(newTask); err != nil {
			output.Error(err)
		}

		output.Success("task created", map[string]any{
			"id": newTask.ID(),
		})
		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&createTitle, "title", "", "Task title (required)")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Task description")
	createCmd.Flags().StringVar(&createStatus, "status", "todo", "Task status (todo, in-progress, done)")
	createCmd.Flags().IntVar(&createPriority, "priority", 0, "Task priority (0=none, 1=urgent, 2=high, 3=normal, 4=low)")
	createCmd.MarkFlagRequired("title")
}
