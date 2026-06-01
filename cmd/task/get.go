package task

import (
	"errors"
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var getPretty bool

var getCmd = &cobra.Command{
	Use:     "get <id>",
	GroupID: "manage",
	Short:   "Get a single task by ID",
	Long:    `Outputs a single task. Use --pretty for human-readable output.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		return cmdutil.WithTaskService(func(svc *task.Service) error {
			t, err := svc.GetTaskByID(taskID)
			if err != nil {
				if errors.Is(err, task.ErrTaskNotFound) {
					return output.ErrorMsgWithCode(
						"task not found: "+taskID,
						output.ErrCodeTaskNotFound,
						"Use pace task list to see available task IDs",
					)
				}
				return output.Error(err)
			}
			if getPretty {
				fmt.Println()
				fmt.Println(formatTaskPretty(*t))
				if d := t.Description(); d != "" {
					fmt.Println("  " + d)
				}
				fmt.Println()
				return nil
			}
			output.JSON(t.ToJSON())
			return nil
		})
	},
}

func init() {
	getCmd.Flags().BoolVar(&getPretty, "pretty", false, "Human-readable formatted output")
}
