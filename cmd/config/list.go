package config

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config entries",
	Long:  `Lists all configuration key-value pairs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.NewDB()
		if err != nil {
			output.Error(err)
		}
		defer db.Close()

		config, err := db.GetAllConfig()
		if err != nil {
			output.Error(err)
		}

		output.Success("config list", map[string]any{
			"config": config,
			"count":  len(config),
		})
		return nil
	},
}
