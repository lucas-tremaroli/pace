package epic

import (
	"errors"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:     "spec",
	GroupID: "manage",
	Short:   "Get or set an epic's spec sections",
	Long:    `Read or edit the structured spec (current/target/constraints/exclusions) plus a freeform fallback.`,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var specGetPretty bool

var specGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Print an epic's spec",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithEpicService(func(svc *epic.Service) error {
			e, err := svc.GetEpicByID(args[0])
			if err != nil {
				return epicLookupError(err, args[0])
			}
			if specGetPretty {
				printEpicDetail(*e)
				return nil
			}
			output.JSON(e.Spec())
			return nil
		})
	},
}

var (
	specCurrent    string
	specTarget     string
	specConstraint string
	specExclusion  string
	specFreeform   string
)

var specSetCmd = &cobra.Command{
	Use:   "set [id]",
	Short: "Set one or more spec sections",
	Long:  `Sets the provided spec sections on an epic. Only flags you pass are changed.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithEpicService(func(svc *epic.Service) error {
			e, err := svc.GetEpicByID(args[0])
			if err != nil {
				return epicLookupError(err, args[0])
			}

			changed := false
			if cmd.Flags().Changed("current-state") {
				e.SetCurrentState(specCurrent)
				changed = true
			}
			if cmd.Flags().Changed("target-state") {
				e.SetTargetState(specTarget)
				changed = true
			}
			if cmd.Flags().Changed("constraints") {
				e.SetConstraints(specConstraint)
				changed = true
			}
			if cmd.Flags().Changed("exclusions") {
				e.SetExclusions(specExclusion)
				changed = true
			}
			if cmd.Flags().Changed("freeform") {
				e.SetFreeform(specFreeform)
				changed = true
			}
			if !changed {
				return output.ErrorMsgWithCode("no spec sections provided", output.ErrCodeInvalidParams, "Pass at least one of --current-state, --target-state, --constraints, --exclusions, --freeform")
			}

			if err := svc.UpdateEpic(*e); err != nil {
				return output.Error(err)
			}
			output.Success("epic spec updated", e.ToJSON())
			return nil
		})
	},
}

func epicLookupError(err error, id string) error {
	if errors.Is(err, epic.ErrEpicNotFound) {
		return output.ErrorMsgWithCode("epic not found: "+id, output.ErrCodeEpicNotFound, "Use pace epic list to see available epic IDs")
	}
	return output.Error(err)
}

func init() {
	specGetCmd.Flags().BoolVar(&specGetPretty, "pretty", false, "Human-readable formatted output")

	specSetCmd.Flags().StringVar(&specCurrent, "current-state", "", "Current state section (markdown)")
	specSetCmd.Flags().StringVar(&specTarget, "target-state", "", "Target state section (markdown)")
	specSetCmd.Flags().StringVar(&specConstraint, "constraints", "", "Constraints section (markdown)")
	specSetCmd.Flags().StringVar(&specExclusion, "exclusions", "", "Exclusions section (markdown)")
	specSetCmd.Flags().StringVar(&specFreeform, "freeform", "", "Freeform fallback section (markdown)")

	specCmd.AddCommand(specGetCmd)
	specCmd.AddCommand(specSetCmd)
}
