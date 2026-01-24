package task

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var depCmd = &cobra.Command{
	Use:   "dep",
	Short: "Manage task dependencies",
	Long:  `Manage blocking relationships between tasks.`,
}

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
			output.Error(err)
		}
		defer svc.Close()

		if err := svc.AddDependency(blockerID, blockedID); err != nil {
			output.Error(err)
		}

		output.Success("dependency added", map[string]any{
			"blocker": blockerID,
			"blocked": blockedID,
		})
		return nil
	},
}

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
			output.Error(err)
		}
		defer svc.Close()

		if err := svc.RemoveDependency(blockerID, blockedID); err != nil {
			output.Error(err)
		}

		output.Success("dependency removed", map[string]any{
			"blocker": blockerID,
			"blocked": blockedID,
		})
		return nil
	},
}

var depListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List dependencies for a task",
	Long:  `Shows what tasks block the given task and what tasks it blocks.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		t, err := svc.GetTaskByID(taskID)
		if err != nil {
			output.Error(err)
		}

		output.JSON(map[string]any{
			"task_id":    taskID,
			"blocked_by": t.BlockedBy(),
			"blocks":     t.Blocks(),
		})
		return nil
	},
}

func init() {
	depCmd.AddCommand(depAddCmd)
	depCmd.AddCommand(depRemoveCmd)
	depCmd.AddCommand(depListCmd)
}
