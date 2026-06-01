package task

import (
	"fmt"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var depChainCmd = &cobra.Command{
	Use:   "chain <id1> <id2> [id3] ...",
	Short: "Create a chain of dependencies",
	Long: `Creates sequential dependencies between tasks.
The first task blocks the second, the second blocks the third, and so on.

Example:
  pace task dep chain pace-001 pace-002 pace-003
  Creates: pace-001 blocks pace-002, pace-002 blocks pace-003`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := task.NewService()
		if err != nil {
			return output.Error(err)
		}
		defer svc.Close()

		var dependencies []map[string]string
		var errs []string

		for i := 0; i < len(args)-1; i++ {
			blockerID := args[i]
			blockedID := args[i+1]

			if err := svc.AddDependency(blockerID, blockedID); err != nil {
				errs = append(errs, fmt.Sprintf("%s->%s: %s", blockerID, blockedID, err.Error()))
			} else {
				dependencies = append(dependencies, map[string]string{
					"blocker": blockerID,
					"blocked": blockedID,
				})
			}
		}

		if len(errs) > 0 && len(dependencies) == 0 {
			return output.ErrorMsg(strings.Join(errs, "; "))
		}

		data := map[string]any{
			"dependencies": dependencies,
		}
		if len(errs) > 0 {
			data["errors"] = errs
		}

		output.Success("dependency chain created", data)
		return nil
	},
}
