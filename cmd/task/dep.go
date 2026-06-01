package task

import (
	"github.com/spf13/cobra"
)

var depCmd = &cobra.Command{
	Use:     "dep",
	GroupID: "workflow",
	Short:   "Manage task dependencies",
	Long:    `Manage blocking relationships between tasks.`,
}

func init() {
	depCmd.AddCommand(depAddCmd)
	depCmd.AddCommand(depRemoveCmd)
	depCmd.AddCommand(depListCmd)
	depCmd.AddCommand(depTreeCmd)
	depCmd.AddCommand(depChainCmd)

	depTreeCmd.Flags().StringVar(&treeDirection, "direction", "up", "Tree direction: 'up' (blockers), 'down' (blocks), or 'both'")
	depTreeCmd.Flags().StringVar(&treeStatus, "status", "", "Filter by status (todo, in-progress, done)")
	depTreeCmd.Flags().IntVarP(&treeMaxDepth, "max-depth", "d", 50, "Maximum tree depth to display")
}
