package tick

import (
	"github.com/charmbracelet/huh"
	"github.com/lucas-tremaroli/pace/internal/tick"
	"github.com/spf13/cobra"
)

var TickCmd = &cobra.Command{
	Use:   "tick",
	Short: "Start a timer for flow state",
	Long:  `Start a focus timer to help you enter a flow state for deep work sessions.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		minutes, _ := cmd.Flags().GetInt("minutes")

		var goal string
		err := huh.NewInput().
			Title("What's your goal for this session?").
			Placeholder("optional").
			CharLimit(50).
			Value(&goal).
			Run()
		if err != nil {
			return err
		}

		svc := tick.NewService(minutes, goal)
		svc.Start()
		return nil
	},
}

func init() {
	TickCmd.GroupID = "recharge"
	TickCmd.Flags().IntP("minutes", "m", 25, "Duration of the focus timer in minutes")
}
