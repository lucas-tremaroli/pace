package config

import (
	"errors"
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Long:  `Retrieves a configuration value by key.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.NewDB()
		if err != nil {
			return output.Error(err)
		}
		defer db.Close()

		key := args[0]
		value, err := db.GetConfig(key)
		if errors.Is(err, storage.ErrNotFound) {
			return output.ErrorMsg(fmt.Sprintf("config key '%s' not found", key))
		} else if err != nil {
			return output.Error(err)
		}

		output.Success("config retrieved", map[string]string{
			"key":   key,
			"value": value,
		})
		return nil
	},
}
