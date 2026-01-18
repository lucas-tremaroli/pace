package focus

import (
	"github.com/lucas-tremaroli/pace/internal/focus"
	"github.com/spf13/cobra"
)

var FocusCmd = &cobra.Command{
	Use:   "focus",
	Short: "Start a simple timer for focus mode",
	Long:  `Starts a simple timer to help you focus for a set period of time.`,
	Run: func(cmd *cobra.Command, args []string) {
		minutes, _ := cmd.Flags().GetInt("minutes")
		svc := focus.NewService(minutes)
		svc.Start()
	},
}

func init() {
	FocusCmd.Flags().IntP("minutes", "m", 25, "Duration of the focus timer in minutes")
}
