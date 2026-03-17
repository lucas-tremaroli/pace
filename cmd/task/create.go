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
	createPriority    int
	createLabel       string
	createLink        string
	createBulk        string
)

var createCmd = &cobra.Command{
	Use:     "create",
	GroupID: "manage",
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
			output.ErrorMsgWithCode("title is required", output.ErrCodeMissingField, "Provide a --title flag")
		}

		status, err := task.ParseStatus(createStatus)
		if err != nil {
			output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: todo, in-progress, done")
		}

		if err := task.ValidateLabel(createLabel); err != nil {
			output.ErrorWithCode(err, output.ErrCodeInvalidType, "Valid values: task, bug, feature, chore, docs")
		}

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		newTask := task.NewTaskComplete(svc.GenerateTaskID(), status, createTitle, createDescription, createPriority, createLink)

		if err := svc.CreateTask(newTask); err != nil {
			output.Error(err)
		}

		if err := svc.SetLabel(newTask.ID(), createLabel); err != nil {
			output.Error(err)
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

		// Default label to "task" if not specified, validate
		label := input.Label
		if label == "" {
			label = "task"
		}
		if err := task.ValidateLabel(label); err != nil {
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

		newTask := task.NewTaskComplete(svc.GenerateTaskID(), status, input.Title, input.Description, priority, input.Link)

		if err := svc.CreateTask(newTask); err != nil {
			result.Failed = append(result.Failed, output.BulkItem{
				Title: input.Title,
				Error: err.Error(),
			})
			continue
		}

		// Set label
		var warnings []string
		if err := svc.SetLabel(newTask.ID(), label); err != nil {
			warnings = append(warnings, "set label '"+label+"': "+err.Error())
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
	createCmd.Flags().IntVar(&createPriority, "priority", 3, "Task priority (1=urgent, 2=high, 3=normal, 4=low)")
	createCmd.Flags().StringVar(&createLabel, "label", "task", "Task label (task, bug, feature, chore, docs)")
	createCmd.Flags().StringVar(&createLink, "url", "", "URL associated with the task (e.g., google.com)")
	createCmd.Flags().StringVar(&createBulk, "bulk", "", "JSON array of tasks to create, or '-' for stdin")
}
