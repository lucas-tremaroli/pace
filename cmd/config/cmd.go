package config

import (
	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage project-specific configuration",
	Long:  `Manage configuration settings for the current pace storage.`,
}

func init() {
	ConfigCmd.GroupID = "configuration"
	ConfigCmd.AddCommand(setCmd)
	ConfigCmd.AddCommand(getCmd)
	ConfigCmd.AddCommand(listCmd)
	ConfigCmd.AddCommand(unsetCmd)
}
