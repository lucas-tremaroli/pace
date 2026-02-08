package task

import (
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	updateTitle        string
	updateDescription  string
	updateStatus       string
	updateType         string
	updatePriority     int
	updateAddLabels    []string
	updateRemoveLabels []string
	updateLink         string
	updateFilters      []string
	updateDryRun       bool
)

var updateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update an existing task or batch update tasks",
	Long: `Updates a task and outputs the result in JSON format. Only specified fields are updated.

For batch updates, use --filter with update flags:
  pace task update --filter status=todo --priority 1
  pace task update --filter type=bug --priority 1 --status in-progress
  pace task update --filter label=sprint-1 --status done --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for conflicting options
		if len(updateFilters) > 0 && len(args) > 0 {
			output.ErrorMsg("cannot use both task ID and --filter (use one or the other)")
		}

		// Check if batch update mode
		if len(updateFilters) > 0 {
			return handleBatchUpdate(cmd)
		}

		// Single task update requires an ID
		if len(args) == 0 {
			output.ErrorMsg("task ID required (or use --filter for batch updates)")
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
			output.Error(err)
		}

		// Apply updates only for flags that were explicitly set
		title := existingTask.Title()
		description := existingTask.Description()
		status := existingTask.Status()
		taskType := existingTask.Type()
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
				output.Error(err)
			}
			status = parsedStatus
		}
		if cmd.Flags().Changed("type") {
			parsedType, err := task.ParseTaskType(updateType)
			if err != nil {
				output.Error(err)
			}
			taskType = parsedType
		}
		if cmd.Flags().Changed("priority") {
			priority = updatePriority
		}
		if cmd.Flags().Changed("url") {
			link = updateLink
		}

		updatedTask := task.NewTaskComplete(taskID, status, taskType, title, description, priority, link)

		if err := svc.UpdateTask(updatedTask); err != nil {
			output.Error(err)
		}

		// Add labels if specified
		for _, label := range updateAddLabels {
			if err := svc.AddLabel(taskID, label); err != nil {
				output.Error(err)
			}
		}

		// Remove labels if specified
		for _, label := range updateRemoveLabels {
			if err := svc.RemoveLabel(taskID, label); err != nil {
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
		output.ErrorMsg("--title, --description, and --url cannot be used with --filter (would set same value for all matched tasks)")
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
	var batchType *task.TaskType
	var batchPriority *int

	if cmd.Flags().Changed("status") {
		parsedStatus, err := task.ParseStatus(updateStatus)
		if err != nil {
			output.Error(err)
		}
		batchStatus = &parsedStatus
	}
	if cmd.Flags().Changed("type") {
		parsedType, err := task.ParseTaskType(updateType)
		if err != nil {
			output.Error(err)
		}
		batchType = &parsedType
	}
	if cmd.Flags().Changed("priority") {
		batchPriority = &updatePriority
	}

	// Validate we have something to update
	if batchStatus == nil && batchType == nil && batchPriority == nil &&
		len(updateAddLabels) == 0 && len(updateRemoveLabels) == 0 {
		output.ErrorMsg("no updates specified (use --status, --type, --priority, --label, or --remove-label)")
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
			if batchType != nil {
				changes["type"] = fmt.Sprintf("%s -> %s", t.Type().String(), batchType.String())
			}
			if batchPriority != nil {
				changes["priority"] = fmt.Sprintf("%d -> %d", t.Priority(), *batchPriority)
			}
			if len(updateAddLabels) > 0 {
				changes["add_labels"] = updateAddLabels
			}
			if len(updateRemoveLabels) > 0 {
				changes["remove_labels"] = updateRemoveLabels
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
		taskType := t.Type()
		priority := t.Priority()

		if batchStatus != nil {
			status = *batchStatus
		}
		if batchType != nil {
			taskType = *batchType
		}
		if batchPriority != nil {
			priority = *batchPriority
		}

		updatedTask := task.NewTaskComplete(t.ID(), status, taskType, t.Title(), t.Description(), priority, t.Link())

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

		// Add labels
		for _, label := range updateAddLabels {
			if err := svc.AddLabel(t.ID(), label); err != nil {
				warnings = append(warnings, "add label '"+label+"': "+err.Error())
			}
		}

		// Remove labels
		for _, label := range updateRemoveLabels {
			if err := svc.RemoveLabel(t.ID(), label); err != nil {
				warnings = append(warnings, "remove label '"+label+"': "+err.Error())
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
	updateCmd.Flags().StringVar(&updateType, "type", "", "Task type (task, bug, feature, chore, docs)")
	updateCmd.Flags().IntVar(&updatePriority, "priority", 0, "Task priority (0=none, 1=urgent, 2=high, 3=normal, 4=low)")
	updateCmd.Flags().StringSliceVar(&updateAddLabels, "label", nil, "Add labels (can be specified multiple times)")
	updateCmd.Flags().StringSliceVar(&updateRemoveLabels, "remove-label", nil, "Remove labels (can be specified multiple times)")
	updateCmd.Flags().StringVar(&updateLink, "url", "", "URL associated with the task (e.g., google.com)")
	updateCmd.Flags().StringArrayVar(&updateFilters, "filter", nil, "Filter tasks to update (status=X, type=X, priority=X, label=X)")
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Preview changes without applying them")
}
