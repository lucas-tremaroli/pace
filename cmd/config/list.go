package config

import (
	"fmt"
	"sort"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
)

var listPretty bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config entries",
	Long:  `Lists all configuration key-value pairs. Use --pretty for human-readable output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.NewDB()
		if err != nil {
			return output.Error(err)
		}
		defer db.Close()

		config, err := db.GetAllConfig()
		if err != nil {
			return output.Error(err)
		}

		if listPretty {
			fmt.Println()
			if len(config) == 0 {
				fmt.Println(dimStyle.Render("No config entries."))
				fmt.Println()
				return nil
			}
			keys := make([]string, 0, len(config))
			for k := range config {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("%s %s\n", keyStyle.Render(k+":"), valueStyle.Render(config[k]))
			}
			fmt.Println()
			fmt.Println(dimStyle.Render(fmt.Sprintf("%d entry(ies)", len(config))))
			return nil
		}

		output.Success("config list", map[string]any{
			"config": config,
			"count":  len(config),
		})
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listPretty, "pretty", false, "Human-readable formatted output")
}
