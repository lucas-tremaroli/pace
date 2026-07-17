package epic

import (
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var (
	listPretty       bool
	listFilterStatus string
)

var listCmd = &cobra.Command{
	Use:     "list",
	GroupID: "manage",
	Short:   "List all epics",
	Long:    `Outputs all epics. Use --pretty for human-readable format, --status to filter.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithEpicService(func(svc *epic.Service) error {
			epics, err := svc.LoadAllEpics()
			if err != nil {
				return output.Error(err)
			}

			if listFilterStatus != "" {
				status, err := epic.ParseStatus(listFilterStatus)
				if err != nil {
					return output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: planning, active, done")
				}
				var filtered []epic.Epic
				for _, e := range epics {
					if e.Status() == status {
						filtered = append(filtered, e)
					}
				}
				epics = filtered
			}

			if listPretty {
				printEpicsPretty(epics)
				return nil
			}

			jsons := make([]epic.EpicJSON, len(epics))
			for i, e := range epics {
				jsons[i] = e.ToJSON()
			}
			output.JSON(map[string]any{"epics": jsons, "count": len(jsons)})
			return nil
		})
	},
}

func printEpicsPretty(epics []epic.Epic) {
	if len(epics) == 0 {
		fmt.Println(dimStyle.Render("No epics found."))
		return
	}
	fmt.Println()
	for _, e := range epics {
		line := idStyle.Render(e.ID()) + " " + titleStyle.Render(e.Title()) + " " + labelStyle.Render("["+e.Status().String()+"]")
		if e.Summary() != "" {
			line += " " + dimStyle.Render("— "+e.Summary())
		}
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("%d epic(s)", len(epics))))
}

func init() {
	listCmd.Flags().BoolVar(&listPretty, "pretty", false, "Human-readable formatted output")
	listCmd.Flags().StringVar(&listFilterStatus, "status", "", "Filter by status (planning, active, done)")
}
