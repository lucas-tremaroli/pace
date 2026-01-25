package cmd

import (
	"os"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
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

		output.Success("initialized project storage", map[string]any{
			"path": paceDir,
		})
		return nil
	},
}

func init() {
	initCmd.GroupID = "configuration"
	rootCmd.AddCommand(initCmd)
}
