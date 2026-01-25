package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
)

var (
	noGitignore bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project-specific pace storage",
	Long: `Creates a .pace/ directory in the current working directory for project-specific storage.

This allows you to have separate tasks and notes for each project, instead of using
the global ~/.config/pace/ storage.

The command will:
  - Create .pace/ directory in the current directory
  - Create .pace/notes/ subdirectory for project notes
  - Add .pace/ to .gitignore if present (skip with --no-gitignore)
  - Report if already initialized (searches upward for existing .pace/)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			output.Error(err)
		}

		// Check if already initialized (search upward)
		existing := storage.FindExistingProjectDir(cwd)
		if existing != "" {
			output.Success("already initialized", map[string]any{
				"path": existing,
			})
			return nil
		}

		// Initialize new project directory
		paceDir, err := storage.InitProjectDir(cwd)
		if err != nil {
			output.Error(err)
		}

		// Handle .gitignore
		gitignoreUpdated := false
		if !noGitignore {
			updated, err := addToGitignore(cwd, ".pace/")
			if err != nil {
				// Non-fatal: just report in output but don't fail
				output.Success("initialized project storage", map[string]any{
					"path":              paceDir,
					"gitignore_updated": false,
					"gitignore_error":   err.Error(),
				})
				return nil
			}
			gitignoreUpdated = updated
		}

		output.Success("initialized project storage", map[string]any{
			"path":              paceDir,
			"gitignore_updated": gitignoreUpdated,
		})
		return nil
	},
}

// addToGitignore adds the specified pattern to .gitignore if not already present.
// Returns true if the file was updated, false if pattern already exists or file doesn't exist.
func addToGitignore(dir, pattern string) (bool, error) {
	gitignorePath := filepath.Join(dir, ".gitignore")

	// Check if .gitignore exists
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return false, nil
	}

	// Read existing content
	file, err := os.Open(gitignorePath)
	if err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(file)
	patternExists := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == pattern || line == strings.TrimSuffix(pattern, "/") {
			patternExists = true
			break
		}
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return false, err
	}

	// Pattern already exists
	if patternExists {
		return false, nil
	}

	// Append pattern to .gitignore
	file, err = os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Check if file ends with newline
	info, err := file.Stat()
	if err != nil {
		return false, err
	}

	needsNewline := false
	if info.Size() > 0 {
		// Read last byte
		lastByte := make([]byte, 1)
		_, err := file.ReadAt(lastByte, info.Size()-1)
		if err == nil && lastByte[0] != '\n' {
			needsNewline = true
		}
	}

	// Write pattern
	if needsNewline {
		if _, err := file.WriteString("\n" + pattern + "\n"); err != nil {
			return false, err
		}
	} else {
		if _, err := file.WriteString(pattern + "\n"); err != nil {
			return false, err
		}
	}

	return true, nil
}

func init() {
	initCmd.GroupID = "configuration"
	initCmd.Flags().BoolVar(&noGitignore, "no-gitignore", false, "Skip adding .pace/ to .gitignore")
	rootCmd.AddCommand(initCmd)
}
