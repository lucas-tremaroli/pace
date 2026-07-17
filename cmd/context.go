package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var contextPretty bool

var (
	ctxHeading = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	ctxDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	ctxLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	ctxID      = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	ctxTitle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Dump active tasks and notes for agent consumption",
	Long:  `Displays storage info, active tasks (todo and in-progress), and notes as structured data. Use --pretty for human-readable output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		taskSvc, err := task.NewService()
		if err != nil {
			return output.Error(err)
		}
		defer taskSvc.Close()

		tasks, err := taskSvc.LoadAllTasks()
		if err != nil {
			return output.Error(err)
		}

		todoStatus := task.Todo
		inProgressStatus := task.InProgress

		todoFilter := task.TaskFilter{Status: &todoStatus}
		inProgressFilter := task.TaskFilter{Status: &inProgressStatus}

		todoTasks := todoFilter.Apply(tasks)
		inProgressTasks := inProgressFilter.Apply(tasks)

		noteSvc, err := note.NewService()
		if err != nil {
			return output.Error(err)
		}

		notes, err := noteSvc.ListNotesWithMeta(false)
		if err != nil {
			return output.Error(err)
		}

		var activeEpics []epic.Epic
		if err := cmdutil.WithEpicService(func(epicSvc *epic.Service) error {
			activeEpics, err = epicSvc.LoadEpicsByStatus(epic.Active)
			if err != nil {
				return output.Error(err)
			}
			return nil
		}); err != nil {
			return err
		}

		resolved, err := storage.ResolvePaceDir()
		if err != nil {
			return output.Error(err)
		}

		if contextPretty {
			printContextPretty(resolved, tasks, todoTasks, inProgressTasks, notes, activeEpics)
			return nil
		}

		todoList := make([]task.TaskJSON, 0, len(todoTasks))
		for _, t := range todoTasks {
			todoList = append(todoList, t.ToJSON())
		}
		inProgressList := make([]task.TaskJSON, 0, len(inProgressTasks))
		for _, t := range inProgressTasks {
			inProgressList = append(inProgressList, t.ToJSON())
		}
		noteList := make([]map[string]any, 0, len(notes))
		for _, n := range notes {
			noteList = append(noteList, map[string]any{
				"filename":    n.Filename,
				"description": n.Description,
				"labels":      n.Labels,
			})
		}

		epicList := make([]map[string]any, 0, len(activeEpics))
		for _, e := range activeEpics {
			epicList = append(epicList, epicSummary(e))
		}

		output.Success("context loaded", map[string]any{
			"storage": map[string]any{
				"path": resolved.Path,
				"type": resolved.Type,
			},
			"active_epics": epicList,
			"tasks": map[string]any{
				"in_progress": inProgressList,
				"todo":        todoList,
			},
			"notes": noteList,
			"summary": map[string]any{
				"tasks": map[string]any{
					"total":       len(tasks),
					"todo":        len(todoList),
					"in_progress": len(inProgressList),
					"done":        len(tasks) - len(todoList) - len(inProgressList),
				},
				"notes": len(noteList),
				"epics": len(epicList),
			},
		})
		return nil
	},
}

// epicSummary returns id/title/status plus first-line spec headlines.
func epicSummary(e epic.Epic) map[string]any {
	spec := e.Spec()
	return map[string]any{
		"id":            e.ID(),
		"title":         e.Title(),
		"status":        e.Status().String(),
		"current_state": firstLine(spec.CurrentState),
		"target_state":  firstLine(spec.TargetState),
	}
}

// firstLine returns the first non-empty line of s, trimmed.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func printContextPretty(resolved storage.ResolvedPath, all, todo, inProgress []task.Task, notes []note.Note, activeEpics []epic.Epic) {
	done := len(all) - len(todo) - len(inProgress)
	fmt.Println()
	fmt.Println(ctxHeading.Render("Storage"))
	fmt.Printf("  %s %s %s\n", ctxLabel.Render("path:"), ctxTitle.Render(resolved.Path), ctxDim.Render("("+string(resolved.Type)+")"))
	fmt.Println()

	if len(activeEpics) > 0 {
		fmt.Println(ctxHeading.Render("Active epics"))
		for _, e := range activeEpics {
			fmt.Println("  " + ctxID.Render(e.ID()) + " " + ctxTitle.Render(e.Title()))
			if h := firstLine(e.Spec().TargetState); h != "" {
				fmt.Println("    " + ctxDim.Render("→ "+h))
			}
		}
		fmt.Println()
	}

	fmt.Println(ctxHeading.Render("Tasks"))
	fmt.Printf("  %s %d total, %d in-progress, %d todo, %d done\n", ctxDim.Render("›"), len(all), len(inProgress), len(todo), done)
	if len(inProgress) > 0 {
		fmt.Println("  " + ctxLabel.Render("in-progress:"))
		for _, t := range inProgress {
			fmt.Println("    " + ctxID.Render(t.ID()) + " " + ctxTitle.Render(t.Title()))
		}
	}
	if len(todo) > 0 {
		fmt.Println("  " + ctxLabel.Render("todo:"))
		for _, t := range todo {
			fmt.Println("    " + ctxID.Render(t.ID()) + " " + ctxTitle.Render(t.Title()))
		}
	}
	fmt.Println()

	fmt.Println(ctxHeading.Render("Notes"))
	fmt.Printf("  %s %d total\n", ctxDim.Render("›"), len(notes))
	for _, n := range notes {
		name := strings.TrimSuffix(n.Filename, ".md")
		line := "    " + ctxTitle.Render(name)
		if n.Description != "" {
			line += " " + ctxDim.Render("— "+n.Description)
		}
		fmt.Println(line)
	}
	fmt.Println()
}

func init() {
	contextCmd.GroupID = "setup"
	contextCmd.Flags().BoolVar(&contextPretty, "pretty", false, "Human-readable formatted output")
	rootCmd.AddCommand(contextCmd)
}
