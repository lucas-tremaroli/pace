package cmd

import (
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show detailed project overview",
	Long: `Displays detailed information about the current pace storage including:
  - Storage path and type
  - Task counts by status
  - Note count
  - Configuration values`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get storage info
		resolved, err := storage.ResolvePaceDir()
		if err != nil {
			output.Error(err)
		}

		// Get task statistics
		taskSvc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer taskSvc.Close()

		tasks, err := taskSvc.LoadAllTasks()
		if err != nil {
			output.Error(err)
		}

		// Count tasks by status
		taskStats := map[string]int{
			"todo":        0,
			"in_progress": 0,
			"done":        0,
		}
		for _, t := range tasks {
			switch t.Status() {
			case task.Todo:
				taskStats["todo"]++
			case task.InProgress:
				taskStats["in_progress"]++
			case task.Done:
				taskStats["done"]++
			}
		}

		// Get note count
		noteSvc, err := note.NewService()
		if err != nil {
			output.Error(err)
		}

		notes, err := noteSvc.ListNotes()
		if err != nil {
			output.Error(err)
		}

		// Get config values
		db, err := storage.NewDB()
		if err != nil {
			output.Error(err)
		}
		defer db.Close()

		config, err := db.GetAllConfig()
		if err != nil {
			output.Error(err)
		}

		output.Success("project info", map[string]any{
			"storage": map[string]any{
				"path": resolved.Path,
				"type": resolved.Type,
			},
			"tasks": map[string]any{
				"total":       len(tasks),
				"todo":        taskStats["todo"],
				"in_progress": taskStats["in_progress"],
				"done":        taskStats["done"],
			},
			"notes": map[string]any{
				"total": len(notes),
			},
			"config": config,
		})
		return nil
	},
}

func init() {
	infoCmd.GroupID = "configuration"
	rootCmd.AddCommand(infoCmd)
}
