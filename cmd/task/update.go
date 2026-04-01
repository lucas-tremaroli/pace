package task

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	updateTitle       string
	updateDescription string
	updateStatus      string
	updatePriority    string
	updateLabel       string
	updateLink        string
	updateFilters     []string
	updateDryRun      bool
)

var updateCmd = &cobra.Command{
	Use:     "update [id]",
	GroupID: "manage",
	Short: "Update an existing task or batch update tasks",
	Long: `Updates a task and outputs the result in JSON format. Only specified fields are updated.

For batch updates, use --filter with update flags:
  pace task update --filter status=todo --priority 1
  pace task update --filter label=bug --status in-progress
  pace task update --filter label=feature --status done --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for conflicting options
		if len(updateFilters) > 0 && len(args) > 0 {
			output.ErrorMsgWithCode("cannot use both task ID and --filter (use one or the other)", output.ErrCodeInvalidParams, "")
		}

		// Check if batch update mode
		if len(updateFilters) > 0 {
			return handleBatchUpdate(cmd)
		}

		// Single task update requires an ID
		if len(args) == 0 {
			output.ErrorMsgWithCode("task ID required (or use --filter for batch updates)", output.ErrCodeMissingField, "")
		}

		taskID := args[0]

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		// Get existing task
		existingTask, err := svc.GetTaskByID(taskID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				output.ErrorMsgWithCode("task not found: "+taskID, output.ErrCodeTaskNotFound, "Use pace task list to see available task IDs")
			}
			output.Error(err)
		}

		// Apply updates only for flags that were explicitly set
		title := existingTask.Title()
		description := existingTask.Description()
		status := existingTask.Status()
		priority := existingTask.Priority()
		link := existingTask.Link()

		if cmd.Flags().Changed("title") {
			title = updateTitle
		}
		if cmd.Flags().Changed("description") {
			description = updateDescription
		}
		if cmd.Flags().Changed("status") {
			parsedStatus, err := task.ParseStatus(updateStatus)
			if err != nil {
				output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: todo, in-progress, done")
			}
			status = parsedStatus
		}
		if cmd.Flags().Changed("priority") {
			parsedPriority, err := task.ParsePriority(updatePriority)
			if err != nil {
				output.ErrorWithCode(err, output.ErrCodeInvalidPriority, task.ValidPriorityHelp)
			}
			priority = parsedPriority
		}
		if cmd.Flags().Changed("url") {
			link = updateLink
		}

		updatedTask := task.NewTaskComplete(taskID, status, title, description, priority, link)

		if err := svc.UpdateTask(updatedTask); err != nil {
			output.Error(err)
		}

		// Set label if specified
		if cmd.Flags().Changed("label") {
			if err := task.ValidateLabel(updateLabel); err != nil {
				output.ErrorWithCode(err, output.ErrCodeInvalidType, "Valid values: task, bug, feature, docs")
			}
			if err := svc.SetLabel(taskID, updateLabel); err != nil {
				output.Error(err)
			}
		}

		// Fetch updated task to include label changes in output
		finalTask, err := svc.GetTaskByID(taskID)
		if err != nil {
			output.Error(err)
		}

		output.Success("task updated", finalTask.ToJSON())
		return nil
	},
}

func handleBatchUpdate(cmd *cobra.Command) error {
	// Reject flags that don't make sense in batch mode
	if cmd.Flags().Changed("title") || cmd.Flags().Changed("description") || cmd.Flags().Changed("url") {
		output.ErrorMsgWithCode("--title, --description, and --url cannot be used with --filter (would set same value for all matched tasks)", output.ErrCodeInvalidParams, "")
	}

	// Parse filters
	var filters []*task.TaskFilter
	for _, f := range updateFilters {
		filter, err := task.ParseFilter(f)
		if err != nil {
			output.Error(err)
		}
		filters = append(filters, filter)
	}
	mergedFilter, err := task.MergeFilters(filters)
	if err != nil {
		output.Error(err)
	}

	// Build update from flags
	var batchStatus *task.Status
	var batchPriority *int

	if cmd.Flags().Changed("status") {
		parsedStatus, err := task.ParseStatus(updateStatus)
		if err != nil {
			output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: todo, in-progress, done")
		}
		batchStatus = &parsedStatus
	}
	if cmd.Flags().Changed("priority") {
		parsedPriority, err := task.ParsePriority(updatePriority)
		if err != nil {
			output.ErrorWithCode(err, output.ErrCodeInvalidPriority, task.ValidPriorityHelp)
		}
		batchPriority = &parsedPriority
	}
	if cmd.Flags().Changed("label") {
		if err := task.ValidateLabel(updateLabel); err != nil {
			output.ErrorWithCode(err, output.ErrCodeInvalidType, "Valid values: task, bug, feature, docs")
		}
	}

	// Validate we have something to update
	if batchStatus == nil && batchPriority == nil && !cmd.Flags().Changed("label") {
		output.ErrorMsgWithCode("no updates specified (use --status, --priority, or --label)", output.ErrCodeInvalidParams, "")
	}

	svc, err := task.NewService()
	if err != nil {
		output.Error(err)
	}
	defer svc.Close()

	// Load all tasks
	tasks, err := svc.LoadAllTasks()
	if err != nil {
		output.Error(err)
	}

	// Filter tasks
	var matchingTasks []task.Task
	for _, t := range tasks {
		if mergedFilter.Matches(t) {
			matchingTasks = append(matchingTasks, t)
		}
	}

	if len(matchingTasks) == 0 {
		output.Success("no tasks matched filter", map[string]any{
			"matched": 0,
		})
		return nil
	}

	// Dry run mode
	if updateDryRun {
		var preview []map[string]any
		for _, t := range matchingTasks {
			changes := make(map[string]any)
			changes["id"] = t.ID()
			changes["title"] = t.Title()
			if batchStatus != nil {
				changes["status"] = fmt.Sprintf("%s -> %s", t.Status().String(), batchStatus.String())
			}
			if batchPriority != nil {
				changes["priority"] = fmt.Sprintf("%d -> %d", t.Priority(), *batchPriority)
			}
			if cmd.Flags().Changed("label") {
				changes["label"] = fmt.Sprintf("%s -> %s", t.Label(), updateLabel)
			}
			preview = append(preview, changes)
		}
		output.Success("dry run - no changes made", map[string]any{
			"matched": len(matchingTasks),
			"preview": preview,
		})
		return nil
	}

	// Apply updates
	result := output.BulkResult{
		Total: len(matchingTasks),
	}

	for _, t := range matchingTasks {
		// Apply changes
		status := t.Status()
		priority := t.Priority()

		if batchStatus != nil {
			status = *batchStatus
		}
		if batchPriority != nil {
			priority = *batchPriority
		}

		updatedTask := task.NewTaskComplete(t.ID(), status, t.Title(), t.Description(), priority, t.Link())

		if err := svc.UpdateTask(updatedTask); err != nil {
			result.Failed = append(result.Failed, output.BulkItem{
				ID:    t.ID(),
				Title: t.Title(),
				Error: err.Error(),
			})
			continue
		}

		// Track warnings for non-fatal label errors
		var warnings []string

		if cmd.Flags().Changed("label") {
			if err := svc.SetLabel(t.ID(), updateLabel); err != nil {
				warnings = append(warnings, "set label '"+updateLabel+"': "+err.Error())
			}
		}

		result.Succeeded = append(result.Succeeded, output.BulkItem{
			ID:       t.ID(),
			Title:    t.Title(),
			Warnings: warnings,
		})
	}

	output.BulkSuccess("tasks updated", result)
	return nil
}

func init() {
	updateCmd.Flags().StringVar(&updateTitle, "title", "", "Task title")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "Task description")
	updateCmd.Flags().StringVar(&updateStatus, "status", "", "Task status (todo, in-progress, done)")
	updateCmd.Flags().StringVar(&updatePriority, "priority", "", "Task priority (high, medium, low or 1-3)")
	updateCmd.Flags().StringVar(&updateLabel, "label", "", "Set task label (task, bug, feature, docs)")
	updateCmd.Flags().StringVar(&updateLink, "url", "", "URL associated with the task (e.g., google.com)")
	updateCmd.Flags().StringArrayVar(&updateFilters, "filter", nil, "Filter tasks to update (status=X, priority=X, label=X)")
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Preview changes without applying them")
}
