package task

import (
	"encoding/json"
	"io"
	"os"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	createTitle       string
	createDescription string
	createStatus      string
	createType        string
	createPriority    int
	createLabels      []string
	createLink        string
	createBulk        string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	Long: `Creates a new task and outputs the result in JSON format.

For bulk creation, use --bulk with a JSON array or '-' for stdin:
  pace task create --bulk '[{"title":"Task 1"},{"title":"Task 2"}]'
  cat tasks.json | pace task create --bulk -`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle bulk creation
		if createBulk != "" {
			return handleBulkCreate(createBulk)
		}

		// Single task creation (existing behavior)
		if createTitle == "" {
			output.ErrorMsg("title is required")
		}

		status, err := task.ParseStatus(createStatus)
		if err != nil {
			output.Error(err)
		}

		taskType, err := task.ParseTaskType(createType)
		if err != nil {
			output.Error(err)
		}

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		newTask := task.NewTaskComplete(svc.GenerateTaskID(), status, taskType, createTitle, createDescription, createPriority, createLink)

		if err := svc.CreateTask(newTask); err != nil {
			output.Error(err)
		}

		// Add labels if specified
		for _, label := range createLabels {
			if err := svc.AddLabel(newTask.ID(), label); err != nil {
				output.Error(err)
			}
		}

		output.Success("task created", map[string]any{
			"id": newTask.ID(),
		})
		return nil
	},
}

func handleBulkCreate(bulkInput string) error {
	var jsonData []byte
	var err error

	if bulkInput == "-" {
		// Read from stdin
		jsonData, err = io.ReadAll(os.Stdin)
		if err != nil {
			output.ErrorMsg("failed to read from stdin: " + err.Error())
		}
	} else {
		jsonData = []byte(bulkInput)
	}

	var inputs []task.TaskInput
	if err := json.Unmarshal(jsonData, &inputs); err != nil {
		output.ErrorMsg("invalid JSON: " + err.Error())
	}

	if len(inputs) == 0 {
		output.ErrorMsg("no tasks provided")
	}

	svc, err := task.NewService()
	if err != nil {
		output.Error(err)
	}
	defer svc.Close()

	result := output.BulkResult{
		Total: len(inputs),
	}

	for _, input := range inputs {
		if input.Title == "" {
			result.Failed = append(result.Failed, output.BulkItem{
				Title: "(empty)",
				Error: "title is required",
			})
			continue
		}

		// Parse status (default to todo)
		statusStr := input.Status
		if statusStr == "" {
			statusStr = "todo"
		}
		status, err := task.ParseStatus(statusStr)
		if err != nil {
			result.Failed = append(result.Failed, output.BulkItem{
				Title: input.Title,
				Error: err.Error(),
			})
			continue
		}

		// Parse type (default to task)
		typeStr := input.Type
		if typeStr == "" {
			typeStr = "task"
		}
		taskType, err := task.ParseTaskType(typeStr)
		if err != nil {
			result.Failed = append(result.Failed, output.BulkItem{
				Title: input.Title,
				Error: err.Error(),
			})
			continue
		}

		// Default priority to 3 (normal) if not specified
		priority := input.Priority
		if priority == 0 {
			priority = 3
		}

		newTask := task.NewTaskComplete(svc.GenerateTaskID(), status, taskType, input.Title, input.Description, priority, input.Link)

		if err := svc.CreateTask(newTask); err != nil {
			result.Failed = append(result.Failed, output.BulkItem{
				Title: input.Title,
				Error: err.Error(),
			})
			continue
		}

		// Add labels if specified, track warnings for failures
		var warnings []string
		for _, label := range input.Labels {
			if err := svc.AddLabel(newTask.ID(), label); err != nil {
				warnings = append(warnings, "add label '"+label+"': "+err.Error())
			}
		}

		result.Succeeded = append(result.Succeeded, output.BulkItem{
			ID:       newTask.ID(),
			Title:    input.Title,
			Warnings: warnings,
		})
	}

	output.BulkSuccess("tasks created", result)
	return nil
}

func init() {
	createCmd.Flags().StringVar(&createTitle, "title", "", "Task title (required for single task creation)")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Task description")
	createCmd.Flags().StringVar(&createStatus, "status", "todo", "Task status (todo, in-progress, done)")
	createCmd.Flags().StringVar(&createType, "type", "task", "Task type (task, bug, feature, chore, docs)")
	createCmd.Flags().IntVar(&createPriority, "priority", 3, "Task priority (1=urgent, 2=high, 3=normal, 4=low)")
	createCmd.Flags().StringSliceVar(&createLabels, "label", nil, "Task labels (can be specified multiple times)")
	createCmd.Flags().StringVar(&createLink, "url", "", "URL associated with the task (e.g., google.com)")
	createCmd.Flags().StringVar(&createBulk, "bulk", "", "JSON array of tasks to create, or '-' for stdin")
}
