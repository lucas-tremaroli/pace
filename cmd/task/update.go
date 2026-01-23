package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	updateTitle       string
	updateDescription string
	updateStatus      string
	updatePriority    int
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an existing task",
	Long:  `Updates a task and outputs the result in JSON format. Only specified fields are updated.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		// Get existing task
		existingTask, err := svc.GetTaskByID(taskID)
		if err != nil {
			output.Error(err)
		}

		// Apply updates only for flags that were explicitly set
		title := existingTask.Title()
		description := existingTask.Description()
		status := existingTask.Status()
		priority := existingTask.Priority()

		if cmd.Flags().Changed("title") {
			title = updateTitle
		}
		if cmd.Flags().Changed("description") {
			description = updateDescription
		}
		if cmd.Flags().Changed("status") {
			parsedStatus, err := task.ParseStatus(updateStatus)
			if err != nil {
				output.Error(err)
			}
			status = parsedStatus
		}
		if cmd.Flags().Changed("priority") {
			priority = updatePriority
		}

		updatedTask := task.NewTaskFull(taskID, status, title, description, priority)

		if err := svc.UpdateTask(updatedTask); err != nil {
			output.Error(err)
		}

		output.Success("task updated", updatedTask.ToJSON())
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateTitle, "title", "", "Task title")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "Task description")
	updateCmd.Flags().StringVar(&updateStatus, "status", "", "Task status (todo, in-progress, done)")
	updateCmd.Flags().IntVar(&updatePriority, "priority", 0, "Task priority (0=none, 1=urgent, 2=high, 3=normal, 4=low)")
}
