package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	content "github.com/lucas-tremaroli/pace/internal/embed"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
)

var (
	noGitignore bool
	noClaudeMD  bool
)

const paceSectionHeader = "## Pace"

var (
	paceSectionContent = content.PaceSection
	claudeMDContent    = "# CLAUDE.md\n\n" + content.PaceSection
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
  - Generate CLAUDE.md for Claude Code (skip with --no-claude-md)
  - Report if already initialized (searches upward for existing .pace/)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return output.Error(err)
		}

		// Check if already initialized (search upward)
		existing := storage.FindExistingProjectDir(cwd)
		if existing != "" {
			// Already initialized, but still try to create/update CLAUDE.md
			claudeMDStatus := ""
			if !noClaudeMD {
				status, err := createOrUpdateClaudeMD(cwd)
				if err == nil {
					claudeMDStatus = status
				}
			}

			data := map[string]any{
				"path": existing,
			}
			if claudeMDStatus != "" {
				data["claude_md"] = claudeMDStatus
			}

			output.Success("already initialized", data)
			return nil
		}

		// Initialize new project directory
		paceDir, err := storage.InitProjectDir(cwd)
		if err != nil {
			return output.Error(err)
		}

		// Handle .gitignore
		gitignoreUpdated := false
		if !noGitignore {
			updated, err := addToGitignore(cwd, ".pace/")
			if err == nil {
				gitignoreUpdated = updated
			}
		}

		// Handle CLAUDE.md
		claudeMDStatus := ""
		if !noClaudeMD {
			status, err := createOrUpdateClaudeMD(cwd)
			if err == nil {
				claudeMDStatus = status
			}
		}

		data := map[string]any{
			"path":              paceDir,
			"gitignore_updated": gitignoreUpdated,
		}
		if claudeMDStatus != "" {
			data["claude_md"] = claudeMDStatus
		}

		output.Success("initialized project storage", data)
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

// createOrUpdateClaudeMD creates or updates CLAUDE.md with the Pace section.
// Returns "created", "updated", or "" (unchanged).
func createOrUpdateClaudeMD(dir string) (string, error) {
	claudeMDPath := filepath.Join(dir, "CLAUDE.md")

	// Check if CLAUDE.md exists
	content, err := os.ReadFile(claudeMDPath)
	if os.IsNotExist(err) {
		// Create new file
		if err := os.WriteFile(claudeMDPath, []byte(claudeMDContent), 0644); err != nil {
			return "", err
		}
		return "created", nil
	} else if err != nil {
		return "", err
	}

	// File exists - check for ## Pace section
	contentStr := string(content)
	before, after, ok := strings.Cut(contentStr, paceSectionHeader)

	if !ok {
		// No Pace section found - append it
		newContent := contentStr
		if !strings.HasSuffix(newContent, "\n") {
			newContent += "\n"
		}
		newContent += "\n" + paceSectionContent

		if err := os.WriteFile(claudeMDPath, []byte(newContent), 0644); err != nil {
			return "", err
		}
		return "updated", nil
	}

	// Pace section exists - find where it ends (next ## or end of file)
	afterPace := after
	nextSectionIdx := strings.Index(afterPace, "\n## ")

	var newContent string
	if nextSectionIdx == -1 {
		// No next section - replace to end of file
		newContent = before + paceSectionContent
	} else {
		// Replace up to next section
		newContent = before + paceSectionContent + afterPace[nextSectionIdx+1:]
	}

	// Check if content actually changed
	if newContent == contentStr {
		return "", nil
	}

	if err := os.WriteFile(claudeMDPath, []byte(newContent), 0644); err != nil {
		return "", err
	}
	return "updated", nil
}

func init() {
	initCmd.GroupID = "setup"
	initCmd.Flags().BoolVar(&noGitignore, "no-gitignore", false, "Skip adding .pace/ to .gitignore")
	initCmd.Flags().BoolVar(&noClaudeMD, "no-claude-md", false, "Skip generating CLAUDE.md")
	rootCmd.AddCommand(initCmd)
}
