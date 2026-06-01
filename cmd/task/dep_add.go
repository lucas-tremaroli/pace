package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var depAddCmd = &cobra.Command{
	Use:   "add <blocker-id> <blocked-id>",
	Short: "Add a dependency (blocker blocks blocked)",
	Long:  `Creates a blocking relationship where the first task blocks the second task.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		blockerID := args[0]
		blockedID := args[1]

		svc, err := task.NewService()
		if err != nil {
			return output.Error(err)
		}
		defer svc.Close()

		if err := svc.AddDependency(blockerID, blockedID); err != nil {
			return output.Error(err)
		}

		output.Success("dependency added", map[string]any{
			"blocker": blockerID,
			"blocked": blockedID,
		})
		return nil
	},
}
