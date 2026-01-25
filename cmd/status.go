package cmd

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current pace storage information",
	Long: `Displays information about the current pace storage location.

Shows:
  - Storage path: The directory where tasks and notes are stored
  - Storage type: Whether using global or project-specific storage`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resolved, err := storage.ResolvePaceDir()
		if err != nil {
			output.Error(err)
		}

		output.Success("storage info", map[string]any{
			"path": resolved.Path,
			"type": resolved.Type,
		})
		return nil
	},
}

func init() {
	statusCmd.GroupID = "configuration"
	rootCmd.AddCommand(statusCmd)
}
