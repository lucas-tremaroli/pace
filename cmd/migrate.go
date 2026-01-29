package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
)

var (
	migrateFrom      string
	migrateTo        string
	migrateDryRun    bool
	migrateTasksOnly bool
	migrateNotesOnly bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Move tasks and notes between storages",
	Long: `Migrate tasks and notes between global (~/.config/pace/) and project-specific (.pace/) storage.

Examples:
  # Move tasks from global to current project
  pace migrate --from global --to project

  # Move tasks from current project to global
  pace migrate --from project --to global

  # Preview what would be migrated
  pace migrate --from global --to project --dry-run

  # Migrate only tasks (not notes)
  pace migrate --from global --to project --tasks-only

  # Migrate only notes (not tasks)
  pace migrate --from global --to project --notes-only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate flags
		if migrateFrom == "" || migrateTo == "" {
			output.ErrorMsg("both --from and --to flags are required")
		}
		if migrateFrom != "global" && migrateFrom != "project" {
			output.ErrorMsg("--from must be 'global' or 'project'")
		}
		if migrateTo != "global" && migrateTo != "project" {
			output.ErrorMsg("--to must be 'global' or 'project'")
		}
		if migrateFrom == migrateTo {
			output.ErrorMsg("--from and --to cannot be the same")
		}
		if migrateTasksOnly && migrateNotesOnly {
			output.ErrorMsg("cannot use both --tasks-only and --notes-only")
		}

		// Get source and destination paths
		globalDir, err := storage.GetGlobalPaceDir()
		if err != nil {
			output.Error(err)
		}

		projectDir, err := storage.GetProjectPaceDir()
		if err != nil {
			output.Error(err)
		}

		if migrateFrom == "project" && projectDir == "" {
			output.ErrorMsg("no project storage found (run 'pace init' first)")
		}
		if migrateTo == "project" && projectDir == "" {
			output.ErrorMsg("no project storage found (run 'pace init' first)")
		}

		var sourceDir, destDir string
		if migrateFrom == "global" {
			sourceDir = globalDir
			destDir = projectDir
		} else {
			sourceDir = projectDir
			destDir = globalDir
		}

		// Check if source exists
		if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
			output.ErrorMsg(fmt.Sprintf("source storage does not exist: %s", sourceDir))
		}

		// Ensure destination exists
		if err := os.MkdirAll(destDir, 0755); err != nil {
			output.Error(err)
		}

		result := map[string]any{
			"from":    migrateFrom,
			"to":      migrateTo,
			"dry_run": migrateDryRun,
		}

		// Migrate tasks
		if !migrateNotesOnly {
			taskResult, err := migrateTasks(sourceDir, destDir, migrateDryRun)
			if err != nil {
				output.Error(err)
			}
			result["tasks"] = taskResult
		}

		// Migrate notes
		if !migrateTasksOnly {
			noteResult, err := migrateNotes(sourceDir, destDir, migrateDryRun)
			if err != nil {
				output.Error(err)
			}
			result["notes"] = noteResult
		}

		if migrateDryRun {
			output.Success("migration preview", result)
		} else {
			output.Success("migration complete", result)
		}
		return nil
	},
}

func migrateTasks(sourceDir, destDir string, dryRun bool) (map[string]any, error) {
	sourceDBPath := filepath.Join(sourceDir, "tasks.db")
	destDBPath := filepath.Join(destDir, "tasks.db")

	// Check if source DB exists
	if _, err := os.Stat(sourceDBPath); os.IsNotExist(err) {
		return map[string]any{
			"migrated":  0,
			"skipped":   0,
			"conflicts": []string{},
		}, nil
	}

	sourceDB, err := storage.NewDBWithPath(sourceDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceDB.Close()

	destDB, err := storage.NewDBWithPath(destDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open destination database: %w", err)
	}
	defer destDB.Close()

	// Get all tasks from source
	sourceTasks, err := sourceDB.GetAllTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to get source tasks: %w", err)
	}

	// Get existing task IDs in destination
	destTasks, err := destDB.GetAllTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to get destination tasks: %w", err)
	}
	destIDs := make(map[string]bool)
	for _, t := range destTasks {
		destIDs[t.ID] = true
	}

	var migrated, skipped int
	var conflicts []string

	for _, task := range sourceTasks {
		if destIDs[task.ID] {
			// ID conflict - skip this task
			conflicts = append(conflicts, task.ID)
			skipped++
			continue
		}

		if !dryRun {
			// Create task in destination
			if err := destDB.CreateTask(task.ID, task.Title, task.Description, task.Status, task.TaskType, task.Priority, task.Link); err != nil {
				return nil, fmt.Errorf("failed to migrate task %s: %w", task.ID, err)
			}

			// Migrate labels for this task
			labels, err := sourceDB.GetLabels(task.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get labels for task %s: %w", task.ID, err)
			}
			for _, label := range labels {
				if err := destDB.AddLabel(task.ID, label); err != nil {
					return nil, fmt.Errorf("failed to migrate label for task %s: %w", task.ID, err)
				}
			}

			// Delete from source
			if err := sourceDB.RemoveAllLabels(task.ID); err != nil {
				return nil, fmt.Errorf("failed to remove labels from source task %s: %w", task.ID, err)
			}
			if err := sourceDB.RemoveAllDependencies(task.ID); err != nil {
				return nil, fmt.Errorf("failed to remove dependencies from source task %s: %w", task.ID, err)
			}
			if err := sourceDB.DeleteTask(task.ID); err != nil {
				return nil, fmt.Errorf("failed to delete source task %s: %w", task.ID, err)
			}
		}
		migrated++
	}

	// Migrate dependencies (only for tasks that were migrated)
	if !dryRun && migrated > 0 {
		blockedByMap, _, err := sourceDB.GetAllDependencies()
		if err != nil {
			return nil, fmt.Errorf("failed to get dependencies: %w", err)
		}

		migratedIDs := make(map[string]bool)
		for _, task := range sourceTasks {
			if !destIDs[task.ID] {
				migratedIDs[task.ID] = true
			}
		}

		for blockedID, blockerIDs := range blockedByMap {
			if !migratedIDs[blockedID] {
				continue
			}
			for _, blockerID := range blockerIDs {
				if migratedIDs[blockerID] {
					if err := destDB.AddDependency(blockerID, blockedID); err != nil {
						return nil, fmt.Errorf("failed to migrate dependency: %w", err)
					}
				}
			}
		}
	}

	return map[string]any{
		"migrated":  migrated,
		"skipped":   skipped,
		"conflicts": conflicts,
	}, nil
}

func migrateNotes(sourceDir, destDir string, dryRun bool) (map[string]any, error) {
	sourceNotesDir := filepath.Join(sourceDir, "notes")
	destNotesDir := filepath.Join(destDir, "notes")

	// Check if source notes dir exists
	if _, err := os.Stat(sourceNotesDir); os.IsNotExist(err) {
		return map[string]any{
			"migrated":  0,
			"skipped":   0,
			"conflicts": []string{},
		}, nil
	}

	// Ensure destination notes dir exists
	if !dryRun {
		if err := os.MkdirAll(destNotesDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create destination notes directory: %w", err)
		}
	}

	// List source notes
	entries, err := os.ReadDir(sourceNotesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read source notes directory: %w", err)
	}

	var migrated, skipped int
	var conflicts []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		sourcePath := filepath.Join(sourceNotesDir, entry.Name())
		destPath := filepath.Join(destNotesDir, entry.Name())

		// Check if file exists in destination
		if _, err := os.Stat(destPath); err == nil {
			conflicts = append(conflicts, entry.Name())
			skipped++
			continue
		}

		if !dryRun {
			// Copy file to destination
			if err := copyFile(sourcePath, destPath); err != nil {
				return nil, fmt.Errorf("failed to copy note %s: %w", entry.Name(), err)
			}

			// Delete from source
			if err := os.Remove(sourcePath); err != nil {
				return nil, fmt.Errorf("failed to delete source note %s: %w", entry.Name(), err)
			}
		}
		migrated++
	}

	return map[string]any{
		"migrated":  migrated,
		"skipped":   skipped,
		"conflicts": conflicts,
	}, nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func init() {
	migrateCmd.GroupID = "configuration"
	migrateCmd.Flags().StringVar(&migrateFrom, "from", "", "Source storage (global or project)")
	migrateCmd.Flags().StringVar(&migrateTo, "to", "", "Destination storage (global or project)")
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Preview migration without making changes")
	migrateCmd.Flags().BoolVar(&migrateTasksOnly, "tasks-only", false, "Migrate only tasks (not notes)")
	migrateCmd.Flags().BoolVar(&migrateNotesOnly, "notes-only", false, "Migrate only notes (not tasks)")
	rootCmd.AddCommand(migrateCmd)
}
