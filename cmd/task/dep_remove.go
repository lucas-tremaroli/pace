package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var depRemoveCmd = &cobra.Command{
	Use:   "remove <blocker-id> <blocked-id>",
	Short: "Remove a dependency",
	Long:  `Removes a blocking relationship between two tasks.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		blockerID := args[0]
		blockedID := args[1]

		svc, err := task.NewService()
		if err != nil {
			return output.Error(err)
		}
		defer svc.Close()

		if err := svc.RemoveDependency(blockerID, blockedID); err != nil {
			return output.Error(err)
		}

		output.Success("dependency removed", map[string]any{
			"blocker": blockerID,
			"blocked": blockedID,
		})
		return nil
	},
}
