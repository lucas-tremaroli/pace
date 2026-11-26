package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
)

var (
	migrateFrom   string
	migrateTo     string
	migrateDryRun bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Move tasks and their epics between storages",
	Long: `Migrate tasks (and the epics they belong to) between global (~/.config/pace/) and project-specific (.pace/) storage.

Examples:
  # Move tasks from global to current project
  pace migrate --from global --to project

  # Move tasks from current project to global
  pace migrate --from project --to global

  # Preview what would be migrated
  pace migrate --from global --to project --dry-run
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate flags
		if migrateFrom == "" || migrateTo == "" {
			return output.ErrorMsg("both --from and --to flags are required")
		}
		if migrateFrom != "global" && migrateFrom != "project" {
			return output.ErrorMsg("--from must be 'global' or 'project'")
		}
		if migrateTo != "global" && migrateTo != "project" {
			return output.ErrorMsg("--to must be 'global' or 'project'")
		}
		if migrateFrom == migrateTo {
			return output.ErrorMsg("--from and --to cannot be the same")
		}

		// Get source and destination paths
		globalDir, err := storage.GetGlobalPaceDir()
		if err != nil {
			return output.Error(err)
		}

		projectDir, err := storage.GetProjectPaceDir()
		if err != nil {
			return output.Error(err)
		}

		if migrateFrom == "project" && projectDir == "" {
			return output.ErrorMsg("no project storage found (run 'pace init' first)")
		}
		if migrateTo == "project" && projectDir == "" {
			return output.ErrorMsg("no project storage found (run 'pace init' first)")
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
			return output.ErrorMsg(fmt.Sprintf("source storage does not exist: %s", sourceDir))
		}

		// Ensure destination exists
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return output.Error(err)
		}

		result := map[string]any{
			"from":    migrateFrom,
			"to":      migrateTo,
			"dry_run": migrateDryRun,
		}

		taskResult, err := migrateTasks(sourceDir, destDir, migrateDryRun)
		if err != nil {
			return output.Error(err)
		}
		result["tasks"] = taskResult

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

	// A preview must not touch either store: opening with the default
	// constructor would convert legacy notes (and delete the note files).
	open := storage.NewDBWithPath
	if dryRun {
		open = storage.NewDBWithPathNoMigration
	}

	sourceDB, err := open(sourceDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceDB.Close()

	destDB, err := open(destDBPath)
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
	epicIDs := map[string]string{} // source epic ID -> destination epic ID

	for _, task := range sourceTasks {
		if destIDs[task.ID] {
			// ID conflict - skip this task
			conflicts = append(conflicts, task.ID)
			skipped++
			continue
		}

		if !dryRun {
			// Bring the task's epic along so epic_id doesn't dangle
			destEpicID := task.EpicID
			if task.EpicID != "" {
				var err error
				if destEpicID, err = migrateEpic(sourceDB, destDB, task.EpicID, epicIDs); err != nil {
					return nil, fmt.Errorf("failed to migrate epic %s for task %s: %w", task.EpicID, task.ID, err)
				}
			}

			// Create task in destination
			if err := destDB.CreateTask(task.ID, task.Title, task.Description, task.Status, task.Priority, task.Link, destEpicID); err != nil {
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

	result := map[string]any{
		"migrated":  migrated,
		"skipped":   skipped,
		"conflicts": conflicts,
	}
	if remapped := remappedEpics(epicIDs); len(remapped) > 0 {
		result["epics_remapped"] = remapped
	}
	return result, nil
}

// migrateEpic makes sure the source epic srcID exists in the destination and
// returns the ID tasks should point at there. Epic IDs are generated per
// store, so the destination may already hold a different epic under the same
// ID: that one is left alone and the source epic is copied under a fresh ID.
// An identical epic is reused. Results are memoised in epicIDs.
func migrateEpic(sourceDB, destDB *storage.DB, srcID string, epicIDs map[string]string) (string, error) {
	if id, ok := epicIDs[srcID]; ok {
		return id, nil
	}
	src, err := sourceDB.GetEpicByID(srcID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil // dangling link in the source: carry the task over unlinked
	}
	if err != nil {
		return "", err
	}

	newID := srcID
	existing, err := destDB.GetEpicByID(srcID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// free in the destination: copy under the same ID
	case err != nil:
		return "", err
	case sameEpic(*src, *existing):
		epicIDs[srcID] = srcID
		return srcID, nil
	default:
		for {
			newID = epic.GenerateID()
			if _, err := destDB.GetEpicByID(newID); errors.Is(err, sql.ErrNoRows) {
				break
			} else if err != nil {
				return "", err
			}
		}
	}

	rec := *src
	rec.ID = newID
	if err := destDB.CreateEpic(rec); err != nil {
		return "", err
	}
	epicIDs[srcID] = newID
	return newID, nil
}

func sameEpic(a, b storage.EpicRecord) bool {
	a.CreatedAt, b.CreatedAt = "", ""
	return a == b
}

// remappedEpics lists the epics that had to change ID in the destination.
func remappedEpics(epicIDs map[string]string) map[string]string {
	out := map[string]string{}
	for from, to := range epicIDs {
		if from != to {
			out[from] = to
		}
	}
	return out
}

func init() {
	migrateCmd.GroupID = "setup"
	migrateCmd.Flags().StringVar(&migrateFrom, "from", "", "Source storage (global or project)")
	migrateCmd.Flags().StringVar(&migrateTo, "to", "", "Destination storage (global or project)")
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Preview migration without making changes")
	rootCmd.AddCommand(migrateCmd)
}
