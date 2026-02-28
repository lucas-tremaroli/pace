package task

import (
	"database/sql"
	"errors"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single task by ID",
	Long:  `Outputs a single task in JSON format.`,
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
			if errors.Is(err, sql.ErrNoRows) {
				output.ErrorMsgWithCode(
					"task not found: "+taskID,
					output.ErrCodeTaskNotFound,
					"Use pace task list to see available task IDs",
				)
			}
			output.Error(err)
		}

		output.JSON(t.ToJSON())
		return nil
	},
}
