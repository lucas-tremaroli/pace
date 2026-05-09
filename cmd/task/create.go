package task

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	createTitle       string
	createDescription string
	createStatus      string
	createPriority    string
	createLabel       string
	createLink        string
	createBulk        string
)

var createCmd = &cobra.Command{
	Use:     "create",
	GroupID: "manage",
	Short:   "Create a new task",
	Long: `Creates a new task and outputs the result in JSON format.

For bulk creation, use --bulk with a JSON array or '-' for stdin:
  pace task create --bulk '[{"title":"Task 1"},{"title":"Task 2"}]'
  cat tasks.json | pace task create --bulk -`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithTaskService(func(svc *task.Service) error {
			if createBulk != "" {
				return handleBulkCreate(svc, createBulk)
			}

			if createTitle == "" {
				return output.ErrorMsgWithCode("title is required", output.ErrCodeMissingField, "Provide a --title flag")
			}

			status, err := task.ParseStatus(createStatus)
			if err != nil {
				return output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: todo, in-progress, done")
			}

			if err := task.ValidateLabel(createLabel); err != nil {
				return output.ErrorWithCode(err, output.ErrCodeInvalidType, "Valid values: task, bug, feature, docs")
			}

			priority, err := task.ParsePriority(createPriority)
			if err != nil {
				return output.ErrorWithCode(err, output.ErrCodeInvalidPriority, task.ValidPriorityHelp)
			}

			newTask := task.NewTaskComplete(svc.GenerateTaskID(), status, createTitle, createDescription, priority, createLink)

			if err := svc.CreateTask(newTask); err != nil {
				return output.Error(err)
			}

			if err := svc.SetLabel(newTask.ID(), createLabel); err != nil {
				return output.Error(err)
			}

			output.Success("task created", map[string]any{"id": newTask.ID()})
			return nil
		})
	},
}

func handleBulkCreate(svc *task.Service, bulkInput string) error {
	var jsonData []byte
	var err error

	if bulkInput == "-" {
		jsonData, err = io.ReadAll(os.Stdin)
		if err != nil {
			return output.ErrorMsg("failed to read from stdin: " + err.Error())
		}
	} else {
		jsonData = []byte(bulkInput)
	}

	var inputs []task.TaskInput
	if err := json.Unmarshal(jsonData, &inputs); err != nil {
		return output.ErrorMsg("invalid JSON: " + err.Error())
	}

	if len(inputs) == 0 {
		return output.ErrorMsg("no tasks provided")
	}

	result := output.BulkResult{Total: len(inputs)}

	for _, input := range inputs {
		if input.Title == "" {
			result.Failed = append(result.Failed, output.BulkItem{Title: "(empty)", Error: "title is required"})
			continue
		}

		statusStr := input.Status
		if statusStr == "" {
			statusStr = "todo"
		}
		status, err := task.ParseStatus(statusStr)
		if err != nil {
			result.Failed = append(result.Failed, output.BulkItem{Title: input.Title, Error: err.Error()})
			continue
		}

		label := input.Label
		if label == "" {
			label = "task"
		}
		if err := task.ValidateLabel(label); err != nil {
			result.Failed = append(result.Failed, output.BulkItem{Title: input.Title, Error: err.Error()})
			continue
		}

		priority := input.Priority
		if priority == 0 {
			priority = 3
		}
		if priority < 1 || priority > 3 {
			result.Failed = append(result.Failed, output.BulkItem{
				Title: input.Title,
				Error: fmt.Sprintf("priority must be 1-3, got %d", priority),
			})
			continue
		}

		newTask := task.NewTaskComplete(svc.GenerateTaskID(), status, input.Title, input.Description, priority, input.Link)

		if err := svc.CreateTask(newTask); err != nil {
			result.Failed = append(result.Failed, output.BulkItem{Title: input.Title, Error: err.Error()})
			continue
		}

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
	createCmd.Flags().StringVar(&createPriority, "priority", "low", "Task priority (high, medium, low or 1-3)")
	createCmd.Flags().StringVar(&createLabel, "label", "task", "Task label (task, bug, feature, docs)")
	createCmd.Flags().StringVar(&createLink, "url", "", "URL associated with the task (e.g., google.com)")
	createCmd.Flags().StringVar(&createBulk, "bulk", "", "JSON array of tasks to create, or '-' for stdin")
}
