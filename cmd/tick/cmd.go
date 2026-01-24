package tick

import (
	"github.com/lucas-tremaroli/pace/internal/tick"
	"github.com/spf13/cobra"
)

var TickCmd = &cobra.Command{
	Use:   "tick",
	Short: "Start a timer for flow state",
	Long:  `Start a focus timer to help you enter a flow state for deep work sessions.`,
	Run: func(cmd *cobra.Command, args []string) {
		minutes, _ := cmd.Flags().GetInt("minutes")
		svc := tick.NewService(minutes)
		svc.Start()
	},
}

func init() {
	TickCmd.GroupID = "recharge"
	TickCmd.Flags().IntP("minutes", "m", 25, "Duration of the focus timer in minutes")
}
