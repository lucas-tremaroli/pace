package task

import (
	"strings"

	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search tasks by text query",
	Long:  `Full-text search across task titles and descriptions.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.ToLower(args[0])

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		allTasks, err := svc.LoadAllTasks()
		if err != nil {
			output.Error(err)
		}

		var matches []task.TaskJSON
		for _, t := range allTasks {
			titleMatch := strings.Contains(strings.ToLower(t.Title()), query)
			descMatch := strings.Contains(strings.ToLower(t.Description()), query)
			if titleMatch || descMatch {
				matches = append(matches, t.ToJSON())
			}
		}

		output.JSON(map[string]any{
			"query":   args[0],
			"tasks":   matches,
			"count":   len(matches),
		})
		return nil
	},
}

