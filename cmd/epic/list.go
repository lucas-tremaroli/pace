package epic

import (
	"fmt"
	"strings"

	"github.com/lucas-tremaroli/pace/v2/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/v2/internal/epic"
	"github.com/lucas-tremaroli/pace/v2/internal/output"
	"github.com/spf13/cobra"
)

var (
	listPretty       bool
	listFilterStatus string
	listLabels       []string
	listHead         int
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

			if len(listLabels) > 0 {
				var filtered []epic.Epic
				for _, e := range epics {
					for _, l := range listLabels {
						if e.HasLabel(l) {
							filtered = append(filtered, e)
							break
						}
					}
				}
				epics = filtered
			}

			if listHead > 0 && listHead < len(epics) {
				epics = epics[:listHead]
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
		if len(e.Labels()) > 0 {
			line += " " + dimStyle.Render("("+strings.Join(e.Labels(), ", ")+")")
		}
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
	listCmd.Flags().StringArrayVar(&listLabels, "label", nil, "Only epics with this label (repeatable; any match)")
	listCmd.Flags().IntVar(&listHead, "head", 0, "Limit output to the first N epics")
	listCmd.Flags().StringVar(&listFilterStatus, "status", "", "Filter by status (planning, active, done)")
}
