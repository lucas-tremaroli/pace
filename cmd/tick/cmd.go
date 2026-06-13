package tick

import (
	"fmt"
	"strconv"

	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/lucas-tremaroli/pace/internal/tick"
	"github.com/spf13/cobra"
)

var minutesFlag string

var TickCmd = &cobra.Command{
	Use:   "tick",
	Short: "Start a timer for flow state",
	Long:  `Start a focus timer to help you enter a flow state for deep work sessions.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		minutes, err := strconv.Atoi(minutesFlag)
		if err != nil {
			cmd.SilenceUsage = true
			return fmt.Errorf("minutes must be a whole number")
		}
		if minutes < 1 || minutes > 60 {
			cmd.SilenceUsage = true
			return fmt.Errorf("minutes must be between 1 and 60")
		}

		var opts []tick.TaskOption
		var closeTask tick.CloseTaskFunc
		var logTask tick.LogTaskFunc
		if ts, err := task.NewService(); err == nil {
			defer ts.Close()
			if ready, err := ts.GetReadyTasks(); err == nil {
				opts = make([]tick.TaskOption, 0, len(ready))
				for _, t := range ready {
					opts = append(opts, tick.TaskOption{ID: t.ID(), Title: t.Title()})
				}
			}
			closeTask = ts.CloseTask
			logTask = ts.LogEntry
		}

		svc := tick.NewService(minutes, opts, closeTask, logTask)
		svc.Start()
		return nil
	},
}

func init() {
	TickCmd.GroupID = "recharge"
	TickCmd.Flags().StringVarP(&minutesFlag, "minutes", "m", "25", "Duration of the focus timer in minutes (max 60)")
}
