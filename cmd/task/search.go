package task

import (
	"strings"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var searchPretty bool

var searchCmd = &cobra.Command{
	Use:     "search <query>",
	GroupID: "workflow",
	Short:   "Search tasks by text query",
	Long:    `Full-text search across task titles and descriptions. Use --pretty for human-readable output.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.ToLower(args[0])
		return cmdutil.WithTaskService(func(svc *task.Service) error {
			allTasks, err := svc.LoadAllTasks()
			if err != nil {
				return output.Error(err)
			}

			var matched []task.Task
			for _, t := range allTasks {
				titleMatch := strings.Contains(strings.ToLower(t.Title()), query)
				descMatch := strings.Contains(strings.ToLower(t.Description()), query)
				if titleMatch || descMatch {
					matched = append(matched, t)
				}
			}

			if searchPretty {
				printTasksPretty(matched)
				return nil
			}

			matches := make([]task.TaskJSON, 0, len(matched))
			for _, t := range matched {
				matches = append(matches, t.ToJSON())
			}
			output.JSON(map[string]any{
				"query": args[0],
				"tasks": matches,
				"count": len(matches),
			})
			return nil
		})
	},
}

func init() {
	searchCmd.Flags().BoolVar(&searchPretty, "pretty", false, "Human-readable formatted output")
}
