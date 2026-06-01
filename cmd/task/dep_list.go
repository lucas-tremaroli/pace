package task

import (
	"errors"
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var depListPretty bool

var depListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List dependencies for a task",
	Long:  `Shows what tasks block the given task and what tasks it blocks. Use --pretty for human-readable output.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		svc, err := task.NewService()
		if err != nil {
			return output.Error(err)
		}
		defer svc.Close()

		t, err := svc.GetTaskByID(taskID)
		if err != nil {
			if errors.Is(err, task.ErrTaskNotFound) {
				return output.ErrorMsgWithCode("task not found: "+taskID, output.ErrCodeTaskNotFound, "Use pace task list to see available task IDs")
			}
			return output.Error(err)
		}

		if depListPretty {
			fmt.Println()
			fmt.Printf("%s %s\n", idStyle.Render(taskID), titleStyle.Render(t.Title()))
			blocked := t.BlockedBy()
			if len(blocked) > 0 {
				fmt.Println(blockedStyle.Render("BLOCKED BY:"))
				for _, id := range blocked {
					fmt.Println("  " + idStyle.Render(id))
				}
			}
			blocks := t.Blocks()
			if len(blocks) > 0 {
				fmt.Println(depStyle.Render("BLOCKS:"))
				for _, id := range blocks {
					fmt.Println("  " + idStyle.Render(id))
				}
			}
			if len(blocked) == 0 && len(blocks) == 0 {
				fmt.Println(countStyle.Render("No dependencies."))
			}
			fmt.Println()
			return nil
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
	depListCmd.Flags().BoolVar(&depListPretty, "pretty", false, "Human-readable formatted output")
}
