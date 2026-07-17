package task

import (
	"errors"
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
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
	updateEpic        string
	updateFilters     []string
	updateDryRun      bool
)

var updateCmd = &cobra.Command{
	Use:     "update [id]",
	GroupID: "manage",
	Short:   "Update an existing task or batch update tasks",
	Long: `Updates a task and outputs the result in JSON format. Only specified fields are updated.

For batch updates, use --filter with update flags:
  pace task update --filter status=todo --priority 1
  pace task update --filter label=bug --status in-progress
  pace task update --filter label=feature --status done --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(updateFilters) > 0 && len(args) > 0 {
			return output.ErrorMsgWithCode("cannot use both task ID and --filter (use one or the other)", output.ErrCodeInvalidParams, "")
		}
		if len(updateFilters) == 0 && len(args) == 0 {
			return output.ErrorMsgWithCode("task ID required (or use --filter for batch updates)", output.ErrCodeMissingField, "")
		}

		return cmdutil.WithTaskService(func(svc *task.Service) error {
			if len(updateFilters) > 0 {
				return handleBatchUpdate(svc, cmd)
			}

			taskID := args[0]

			existingTask, err := svc.GetTaskByID(taskID)
			if err != nil {
				if errors.Is(err, task.ErrTaskNotFound) {
					return output.ErrorMsgWithCode("task not found: "+taskID, output.ErrCodeTaskNotFound, "Use pace task list to see available task IDs")
				}
				return output.Error(err)
			}

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
					return output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: todo, in-progress, done")
				}
				status = parsedStatus
			}
			if cmd.Flags().Changed("priority") {
				parsedPriority, err := task.ParsePriority(updatePriority)
				if err != nil {
					return output.ErrorWithCode(err, output.ErrCodeInvalidPriority, task.ValidPriorityHelp)
				}
				priority = parsedPriority
			}
			if cmd.Flags().Changed("url") {
				link = updateLink
			}

			updatedTask := task.NewTaskComplete(taskID, status, title, description, priority, link)
			if err := svc.UpdateTask(updatedTask); err != nil {
				return output.Error(err)
			}

			if cmd.Flags().Changed("label") {
				if err := task.ValidateLabel(updateLabel); err != nil {
					return output.ErrorWithCode(err, output.ErrCodeInvalidType, "Valid values: task, bug, feature, docs")
				}
				if err := svc.SetLabel(taskID, updateLabel); err != nil {
					return output.Error(err)
				}
			}

			if cmd.Flags().Changed("epic") {
				if err := svc.SetEpic(taskID, updateEpic); err != nil {
					return output.Error(err)
				}
			}

			finalTask, err := svc.GetTaskByID(taskID)
			if err != nil {
				return output.Error(err)
			}
			output.Success("task updated", finalTask.ToJSON())
			return nil
		})
	},
}

func handleBatchUpdate(svc *task.Service, cmd *cobra.Command) error {
	if cmd.Flags().Changed("title") || cmd.Flags().Changed("description") || cmd.Flags().Changed("url") {
		return output.ErrorMsgWithCode("--title, --description, and --url cannot be used with --filter (would set same value for all matched tasks)", output.ErrCodeInvalidParams, "")
	}

	var filters []*task.TaskFilter
	for _, f := range updateFilters {
		filter, err := task.ParseFilter(f)
		if err != nil {
			return output.Error(err)
		}
		filters = append(filters, filter)
	}
	mergedFilter, err := task.MergeFilters(filters)
	if err != nil {
		return output.Error(err)
	}

	var batchStatus *task.Status
	var batchPriority *int

	if cmd.Flags().Changed("status") {
		parsedStatus, err := task.ParseStatus(updateStatus)
		if err != nil {
			return output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: todo, in-progress, done")
		}
		batchStatus = &parsedStatus
	}
	if cmd.Flags().Changed("priority") {
		parsedPriority, err := task.ParsePriority(updatePriority)
		if err != nil {
			return output.ErrorWithCode(err, output.ErrCodeInvalidPriority, task.ValidPriorityHelp)
		}
		batchPriority = &parsedPriority
	}
	if cmd.Flags().Changed("label") {
		if err := task.ValidateLabel(updateLabel); err != nil {
			return output.ErrorWithCode(err, output.ErrCodeInvalidType, "Valid values: task, bug, feature, docs")
		}
	}

	if batchStatus == nil && batchPriority == nil && !cmd.Flags().Changed("label") && !cmd.Flags().Changed("epic") {
		return output.ErrorMsgWithCode("no updates specified (use --status, --priority, --label, or --epic)", output.ErrCodeInvalidParams, "")
	}

	tasks, err := svc.LoadAllTasks()
	if err != nil {
		return output.Error(err)
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

	if updateDryRun {
		var preview []map[string]any
		for _, t := range matchingTasks {
			changes := map[string]any{"id": t.ID(), "title": t.Title()}
			if batchStatus != nil {
				changes["status"] = fmt.Sprintf("%s -> %s", t.Status().String(), batchStatus.String())
			}
			if batchPriority != nil {
				changes["priority"] = fmt.Sprintf("%d -> %d", t.Priority(), *batchPriority)
			}
			if cmd.Flags().Changed("label") {
				changes["label"] = fmt.Sprintf("%s -> %s", t.Label(), updateLabel)
			}
			if cmd.Flags().Changed("epic") {
				changes["epic"] = fmt.Sprintf("%s -> %s", t.EpicID(), updateEpic)
			}
			preview = append(preview, changes)
		}
		output.Success("dry run - no changes made", map[string]any{
			"matched": len(matchingTasks),
			"preview": preview,
		})
		return nil
	}

	result := output.BulkResult{Total: len(matchingTasks)}
	for _, t := range matchingTasks {
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
			result.Failed = append(result.Failed, output.BulkItem{ID: t.ID(), Title: t.Title(), Error: err.Error()})
			continue
		}

		var warnings []string
		if cmd.Flags().Changed("label") {
			if err := svc.SetLabel(t.ID(), updateLabel); err != nil {
				warnings = append(warnings, "set label '"+updateLabel+"': "+err.Error())
			}
		}
		if cmd.Flags().Changed("epic") {
			if err := svc.SetEpic(t.ID(), updateEpic); err != nil {
				warnings = append(warnings, "set epic '"+updateEpic+"': "+err.Error())
			}
		}
		result.Succeeded = append(result.Succeeded, output.BulkItem{ID: t.ID(), Title: t.Title(), Warnings: warnings})
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
	updateCmd.Flags().StringVar(&updateEpic, "epic", "", "Assign the task to an epic (empty string clears it)")
	updateCmd.Flags().StringArrayVar(&updateFilters, "filter", nil, "Filter tasks to update (status=X, priority=X, label=X, epic=X)")
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Preview changes without applying them")
}
