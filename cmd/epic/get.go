package epic

import (
	"errors"
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var getPretty bool

var getCmd = &cobra.Command{
	Use:     "get [id]",
	GroupID: "manage",
	Short:   "Get a single epic by ID",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithEpicService(func(svc *epic.Service) error {
			e, err := svc.GetEpicByID(args[0])
			if err != nil {
				if errors.Is(err, epic.ErrEpicNotFound) {
					return output.ErrorMsgWithCode("epic not found: "+args[0], output.ErrCodeEpicNotFound, "Use pace epic list to see available epic IDs")
				}
				return output.Error(err)
			}

			if getPretty {
				printEpicDetail(*e)
				return nil
			}
			output.JSON(e.ToJSON())
			return nil
		})
	},
}

func printEpicDetail(e epic.Epic) {
	fmt.Println()
	fmt.Println(headingStyle.Render(e.Title()) + " " + idStyle.Render("("+e.ID()+")") + " " + labelStyle.Render("["+e.Status().String()+"]"))
	if e.Summary() != "" {
		fmt.Println(dimStyle.Render(e.Summary()))
	}
	spec := e.Spec()
	printSection("Current state", spec.CurrentState)
	printSection("Target state", spec.TargetState)
	printSection("Constraints", spec.Constraints)
	printSection("Exclusions", spec.Exclusions)
	printSection("Notes", spec.Freeform)
	fmt.Println()
}

func printSection(name, body string) {
	if body == "" {
		return
	}
	fmt.Println()
	fmt.Println(labelStyle.Render(name+":"))
	fmt.Println(titleStyle.Render(body))
}

func init() {
	getCmd.Flags().BoolVar(&getPretty, "pretty", false, "Human-readable formatted output")
}
