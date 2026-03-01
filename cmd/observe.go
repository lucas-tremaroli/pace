package cmd

import (
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

func init() {
	observeCmd.GroupID = "core"
}

var observeCmd = &cobra.Command{
	Use:   "observe <message>",
	Short: "Record a standalone observation",
	Long:  `Record an observation or learning not tied to a specific task.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := args[0]

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		if err := svc.Observe(message); err != nil {
			output.Error(err)
		}

		output.Success("observation recorded", nil)
		return nil
	},
}
