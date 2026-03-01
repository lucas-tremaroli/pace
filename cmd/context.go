package cmd

import (
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var contextSummary bool

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Dump active tasks and notes for agent consumption",
	Long: `Displays active tasks (todo and in-progress) and notes as structured data.
Use --summary for just counts and stats.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		taskSvc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer taskSvc.Close()

		tasks, err := taskSvc.LoadAllTasks()
		if err != nil {
			output.Error(err)
		}

		if contextSummary {
			return runContextSummary(tasks)
		}

		// Filter to active tasks (todo + in-progress)
		todoStatus := task.Todo
		inProgressStatus := task.InProgress

		todoFilter := task.TaskFilter{Status: &todoStatus}
		inProgressFilter := task.TaskFilter{Status: &inProgressStatus}

		todoTasks := todoFilter.Apply(tasks)
		inProgressTasks := inProgressFilter.Apply(tasks)

		// Build task JSON lists
		todoList := make([]task.TaskJSON, 0, len(todoTasks))
		for _, t := range todoTasks {
			todoList = append(todoList, t.ToJSON())
		}

		inProgressList := make([]task.TaskJSON, 0, len(inProgressTasks))
		for _, t := range inProgressTasks {
			inProgressList = append(inProgressList, t.ToJSON())
		}

		// Load notes
		noteSvc, err := note.NewService()
		if err != nil {
			output.Error(err)
		}

		notes, err := noteSvc.ListNotesWithMeta(false)
		if err != nil {
			output.Error(err)
		}

		noteList := make([]map[string]any, 0, len(notes))
		for _, n := range notes {
			noteList = append(noteList, map[string]any{
				"filename":    n.Filename,
				"description": n.Description,
				"labels":      n.Labels,
			})
		}

		resolved, err := storage.ResolvePaceDir()
		if err != nil {
			output.Error(err)
		}

		output.Success("context loaded", map[string]any{
			"storage": map[string]any{
				"path": resolved.Path,
				"type": resolved.Type,
			},
			"in_progress": inProgressList,
			"todo":        todoList,
			"notes":       noteList,
			"summary": map[string]any{
				"in_progress": len(inProgressList),
				"todo":        len(todoList),
				"notes":       len(noteList),
			},
		})
		return nil
	},
}

func runContextSummary(tasks []task.Task) error {
	var todoCount, inProgressCount, doneCount int
	for _, t := range tasks {
		switch t.Status() {
		case task.Todo:
			todoCount++
		case task.InProgress:
			inProgressCount++
		case task.Done:
			doneCount++
		}
	}

	noteSvc, err := note.NewService()
	if err != nil {
		output.Error(err)
	}

	notes, err := noteSvc.ListNotes()
	if err != nil {
		output.Error(err)
	}

	resolved, err := storage.ResolvePaceDir()
	if err != nil {
		output.Error(err)
	}

	output.Success("project summary", map[string]any{
		"storage": map[string]any{
			"path": resolved.Path,
			"type": resolved.Type,
		},
		"tasks": map[string]any{
			"total":       len(tasks),
			"todo":        todoCount,
			"in_progress": inProgressCount,
			"done":        doneCount,
		},
		"notes": map[string]any{
			"total": len(notes),
		},
	})
	return nil
}

func init() {
	contextCmd.Flags().BoolVar(&contextSummary, "summary", false, "Show only counts and stats")
	contextCmd.GroupID = "configuration"
	rootCmd.AddCommand(contextCmd)
}
