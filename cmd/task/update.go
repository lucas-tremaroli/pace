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
	updateSets         []string
	updateDryRun       bool
)

var updateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update an existing task or batch update tasks",
	Long: `Updates a task and outputs the result in JSON format. Only specified fields are updated.

For batch updates, use --filter and --set:
  pace task update --filter status=todo --set priority=1
  pace task update --filter type=bug --set priority=1 --set status=in-progress
  pace task update --filter label=sprint-1 --set status=done --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
	// Parse filters
	var filters []*task.TaskFilter
	for _, f := range updateFilters {
		filter, err := task.ParseFilter(f)
		if err != nil {
			output.Error(err)
		}
		filters = append(filters, filter)
	}
	mergedFilter := task.MergeFilters(filters)

	// Parse set values
	var updates []*task.TaskUpdate
	for _, s := range updateSets {
		update, err := task.ParseSetValue(s)
		if err != nil {
			output.Error(err)
		}
		updates = append(updates, update)
	}
	mergedUpdate := task.MergeUpdates(updates)

	// Validate we have something to update
	if mergedUpdate.Status == nil && mergedUpdate.Type == nil && mergedUpdate.Priority == nil &&
		len(updateAddLabels) == 0 && len(updateRemoveLabels) == 0 {
		output.ErrorMsg("no updates specified (use --set, --label, or --remove-label)")
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
			if mergedUpdate.Status != nil {
				changes["status"] = fmt.Sprintf("%s -> %s", t.Status().String(), mergedUpdate.Status.String())
			}
			if mergedUpdate.Type != nil {
				changes["type"] = fmt.Sprintf("%s -> %s", t.Type().String(), mergedUpdate.Type.String())
			}
			if mergedUpdate.Priority != nil {
				changes["priority"] = fmt.Sprintf("%d -> %d", t.Priority(), *mergedUpdate.Priority)
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

		if mergedUpdate.Status != nil {
			status = *mergedUpdate.Status
		}
		if mergedUpdate.Type != nil {
			taskType = *mergedUpdate.Type
		}
		if mergedUpdate.Priority != nil {
			priority = *mergedUpdate.Priority
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

		// Add labels
		for _, label := range updateAddLabels {
			if err := svc.AddLabel(t.ID(), label); err != nil {
				// Log error but continue
				result.Failed = append(result.Failed, output.BulkItem{
					ID:    t.ID(),
					Title: t.Title(),
					Error: "add label: " + err.Error(),
				})
			}
		}

		// Remove labels
		for _, label := range updateRemoveLabels {
			if err := svc.RemoveLabel(t.ID(), label); err != nil {
				// Log error but continue
				result.Failed = append(result.Failed, output.BulkItem{
					ID:    t.ID(),
					Title: t.Title(),
					Error: "remove label: " + err.Error(),
				})
			}
		}

		result.Succeeded = append(result.Succeeded, output.BulkItem{
			ID:    t.ID(),
			Title: t.Title(),
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
	updateCmd.Flags().StringArrayVar(&updateFilters, "filter", nil, "Filter tasks (status=X, type=X, priority=X, label=X)")
	updateCmd.Flags().StringArrayVar(&updateSets, "set", nil, "Set field value (status=X, type=X, priority=X)")
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Preview changes without applying them")
}
