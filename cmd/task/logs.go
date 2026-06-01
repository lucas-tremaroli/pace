package task

import (
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var logsPretty bool

var logsCmd = &cobra.Command{
	Use:     "logs <id>",
	GroupID: "logging",
	Short:   "View log entries for a task",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		return cmdutil.WithTaskService(func(svc *task.Service) error {
			logs, err := svc.GetTaskLogs(taskID)
			if err != nil {
				return output.Error(err)
			}
			if logsPretty {
				fmt.Println()
				if len(logs) == 0 {
					fmt.Println(countStyle.Render("No log entries."))
					fmt.Println()
					return nil
				}
				for _, l := range logs {
					prefix := noteStyle.Render(l.CreatedAt)
					if l.Type != "" && l.Type != "log" {
						prefix += " " + labelStyle.Render("["+l.Type+"]")
					}
					fmt.Printf("%s %s\n", prefix, l.Message)
				}
				fmt.Println()
				fmt.Println(countStyle.Render(fmt.Sprintf("%d log(s)", len(logs))))
				return nil
			}
			output.JSON(map[string]any{
				"task_id": taskID,
				"logs":    logs,
				"count":   len(logs),
			})
			return nil
		})
	},
}

func init() {
	logsCmd.Flags().BoolVar(&logsPretty, "pretty", false, "Human-readable formatted output")
}
