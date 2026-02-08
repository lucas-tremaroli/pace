package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	deleteFilters []string
	deleteDryRun  bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete [id] [id2] [id3] ...",
	Short: "Delete one or more tasks by ID or filter",
	Long: `Deletes one or more tasks without confirmation and outputs the result in JSON format.

Delete by ID:
  pace task delete pace-001
  pace task delete pace-001 pace-002 pace-003

Delete by filter:
  pace task delete --filter status=done
  pace task delete --filter type=bug --filter priority=4
  pace task delete --filter label=sprint-1 --dry-run`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for conflicting options
		if len(deleteFilters) > 0 && len(args) > 0 {
			output.ErrorMsg("cannot use both task IDs and --filter (use one or the other)")
		}

		// Check if filter-based deletion
		if len(deleteFilters) > 0 {
			return handleFilterDelete()
		}

		// ID-based deletion requires at least one ID
		if len(args) == 0 {
			output.ErrorMsg("task ID required (or use --filter for filter-based deletion)")
		}

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		// Single ID: backward compatible behavior
		if len(args) == 1 {
			taskID := args[0]
			if err := svc.DeleteTask(taskID); err != nil {
				output.Error(err)
			}
			output.Success("task deleted", map[string]string{
				"id": taskID,
			})
			return nil
		}

		// Multiple IDs: bulk delete
		result := output.BulkResult{
			Total: len(args),
		}

		for _, taskID := range args {
			if err := svc.DeleteTask(taskID); err != nil {
				result.Failed = append(result.Failed, output.BulkItem{
					ID:    taskID,
					Error: err.Error(),
				})
			} else {
				result.Succeeded = append(result.Succeeded, output.BulkItem{
					ID: taskID,
				})
			}
		}

		output.BulkSuccess("tasks deleted", result)
		return nil
	},
}

func handleFilterDelete() error {
	// Parse filters
	var filters []*task.TaskFilter
	for _, f := range deleteFilters {
		filter, err := task.ParseFilter(f)
		if err != nil {
			output.Error(err)
		}
		filters = append(filters, filter)
	}
	mergedFilter := task.MergeFilters(filters)

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
	if deleteDryRun {
		var preview []map[string]any
		for _, t := range matchingTasks {
			preview = append(preview, map[string]any{
				"id":     t.ID(),
				"title":  t.Title(),
				"status": t.Status().String(),
				"type":   t.Type().String(),
			})
		}
		output.Success("dry run - no tasks deleted", map[string]any{
			"matched": len(matchingTasks),
			"preview": preview,
		})
		return nil
	}

	// Delete matching tasks
	result := output.BulkResult{
		Total: len(matchingTasks),
	}

	for _, t := range matchingTasks {
		if err := svc.DeleteTask(t.ID()); err != nil {
			result.Failed = append(result.Failed, output.BulkItem{
				ID:    t.ID(),
				Title: t.Title(),
				Error: err.Error(),
			})
		} else {
			result.Succeeded = append(result.Succeeded, output.BulkItem{
				ID:    t.ID(),
				Title: t.Title(),
			})
		}
	}

	output.BulkSuccess("tasks deleted", result)
	return nil
}

func init() {
	deleteCmd.Flags().StringArrayVar(&deleteFilters, "filter", nil, "Filter tasks to delete (status=X, type=X, priority=X, label=X)")
	deleteCmd.Flags().BoolVar(&deleteDryRun, "dry-run", false, "Preview deletions without applying them")
}
