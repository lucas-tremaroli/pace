package task

import (
	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	deleteFilters []string
	deleteDryRun  bool
)

var deleteCmd = &cobra.Command{
	Use:     "delete [id] [id2] [id3] ...",
	GroupID: "manage",
	Short:   "Delete one or more tasks by ID or filter",
	Long: `Deletes one or more tasks without confirmation and outputs the result in JSON format.

Delete by ID:
  pace task delete pace-001
  pace task delete pace-001 pace-002 pace-003

Delete by filter:
  pace task delete --filter status=done
  pace task delete --filter label=bug --filter priority=4
  pace task delete --filter label=feature --dry-run`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(deleteFilters) > 0 && len(args) > 0 {
			output.ErrorMsgWithCode("cannot use both task IDs and --filter (use one or the other)", output.ErrCodeInvalidParams, "")
		}
		if len(deleteFilters) == 0 && len(args) == 0 {
			output.ErrorMsgWithCode("task ID required (or use --filter for filter-based deletion)", output.ErrCodeMissingField, "")
		}

		return cmdutil.WithTaskService(func(svc *task.Service) error {
			if len(deleteFilters) > 0 {
				return handleFilterDelete(svc)
			}

			if len(args) == 1 {
				taskID := args[0]
				if err := svc.DeleteTask(taskID); err != nil {
					output.Error(err)
				}
				output.Success("task deleted", map[string]string{"id": taskID})
				return nil
			}

			result := output.BulkResult{Total: len(args)}
			for _, taskID := range args {
				if err := svc.DeleteTask(taskID); err != nil {
					result.Failed = append(result.Failed, output.BulkItem{ID: taskID, Error: err.Error()})
				} else {
					result.Succeeded = append(result.Succeeded, output.BulkItem{ID: taskID})
				}
			}
			output.BulkSuccess("tasks deleted", result)
			return nil
		})
	},
}

func handleFilterDelete(svc *task.Service) error {
	var filters []*task.TaskFilter
	for _, f := range deleteFilters {
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

	tasks, err := svc.LoadAllTasks()
	if err != nil {
		output.Error(err)
	}

	var matchingTasks []task.Task
	for _, t := range tasks {
		if mergedFilter.Matches(t) {
			matchingTasks = append(matchingTasks, t)
		}
	}

	if len(matchingTasks) == 0 {
		output.Success("no tasks matched filter", map[string]any{"matched": 0})
		return nil
	}

	if deleteDryRun {
		var preview []map[string]any
		for _, t := range matchingTasks {
			preview = append(preview, map[string]any{
				"id":     t.ID(),
				"title":  t.Title(),
				"status": t.Status().String(),
				"label":  t.Label(),
			})
		}
		output.Success("dry run - no tasks deleted", map[string]any{
			"matched": len(matchingTasks),
			"preview": preview,
		})
		return nil
	}

	result := output.BulkResult{Total: len(matchingTasks)}
	for _, t := range matchingTasks {
		if err := svc.DeleteTask(t.ID()); err != nil {
			result.Failed = append(result.Failed, output.BulkItem{ID: t.ID(), Title: t.Title(), Error: err.Error()})
		} else {
			result.Succeeded = append(result.Succeeded, output.BulkItem{ID: t.ID(), Title: t.Title()})
		}
	}
	output.BulkSuccess("tasks deleted", result)
	return nil
}

func init() {
	deleteCmd.Flags().StringArrayVar(&deleteFilters, "filter", nil, "Filter tasks to delete (status=X, priority=X, label=X)")
	deleteCmd.Flags().BoolVar(&deleteDryRun, "dry-run", false, "Preview deletions without applying them")
}
