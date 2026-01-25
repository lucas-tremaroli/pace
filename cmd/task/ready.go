package task

import (
	"fmt"
	"slices"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var readyPretty bool

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Show tasks ready to work on",
	Long:  `Lists tasks that have no blockers (or all blockers are done). Use --pretty for human-readable format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		tasks, err := svc.GetReadyTasks()
		if err != nil {
			output.Error(err)
		}

		// Sort by priority (P1 first, P4 last)
		slices.SortFunc(tasks, func(a, b task.Task) int {
			return a.Priority() - b.Priority()
		})

		if readyPretty {
			if len(tasks) == 0 {
				fmt.Println(countStyle.Render("No ready tasks."))
				return nil
			}
			for _, t := range tasks {
				fmt.Println(formatTaskPretty(t))
			}
			fmt.Println()
			fmt.Println(countStyle.Render(fmt.Sprintf("%d ready task(s)", len(tasks))))
			printLegend()
			return nil
		}

		var tasksJSON []task.TaskJSON
		for _, t := range tasks {
			tasksJSON = append(tasksJSON, t.ToJSON())
		}

		output.JSON(tasksJSON)
		return nil
	},
}

func init() {
	readyCmd.Flags().BoolVar(&readyPretty, "pretty", false, "Human-readable formatted output")
}
