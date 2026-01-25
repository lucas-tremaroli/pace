package config

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long:  `Sets a configuration value for the current pace storage.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.NewDB()
		if err != nil {
			output.Error(err)
		}
		defer db.Close()

		key := args[0]
		value := args[1]

		if err := db.SetConfig(key, value); err != nil {
			output.Error(err)
		}

		output.Success("config set", map[string]string{
			"key":   key,
			"value": value,
		})
		return nil
	},
}
