package epic

import (
	"io"
	"os"

	"github.com/lucas-tremaroli/pace/v2/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/v2/internal/epic"
	"github.com/lucas-tremaroli/pace/v2/internal/output"
	"github.com/spf13/cobra"
)

var (
	createTitle   string
	createSummary string
	createStatus  string
	createLabels  []string
	createEdit    bool

	createCurrent    string
	createTarget     string
	createConstraint string
	createExclusion  string
	createFreeform   string
)

var createCmd = &cobra.Command{
	Use:     "create",
	GroupID: "manage",
	Short:   "Create a new epic",
	Long: `Creates a new epic and outputs the result in JSON format.
Spec sections can be passed as flags; use --freeform - to read the freeform section from stdin.
Use --edit to write the epic in $EDITOR instead.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithEpicService(func(svc *epic.Service) error {
			if createTitle == "" && !createEdit {
				return output.ErrorMsgWithCode("title is required", output.ErrCodeMissingField, "Provide a --title flag (or --edit)")
			}

			status, err := epic.ParseStatus(createStatus)
			if err != nil {
				return output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: planning, active, done")
			}

			// Reading stdin is opt-in (--freeform -): agent shells often leave
			// stdin as an open pipe, and an implicit read would block forever.
			if createFreeform == "-" {
				piped, err := io.ReadAll(os.Stdin)
				if err != nil {
					return output.Error(err)
				}
				createFreeform = string(piped)
			}

			e := epic.NewEpic(svc.GenerateEpicID(), status, createTitle, createSummary)
			e.SetSpec(epic.Spec{
				CurrentState: createCurrent,
				TargetState:  createTarget,
				Constraints:  createConstraint,
				Exclusions:   createExclusion,
				Freeform:     createFreeform,
			})
			e.SetLabels(createLabels)

			if createEdit {
				if createTitle == "" {
					e.SetTitle("Untitled epic")
				}
				edited, err := cmdutil.EditText(epic.Format(e))
				if err != nil {
					return output.Error(err)
				}
				if e, err = epic.Parse(edited, e); err != nil {
					return output.ErrorWithCode(err, output.ErrCodeInvalidParams, rejectedHint(edited))
				}
			}

			if err := svc.CreateEpic(e); err != nil {
				return output.Error(err)
			}

			output.Success("epic created", map[string]any{"id": e.ID()})
			return nil
		})
	},
}

func init() {
	createCmd.Flags().StringVar(&createTitle, "title", "", "Epic title (required)")
	createCmd.Flags().StringVar(&createSummary, "summary", "", "Short summary of the epic")
	createCmd.Flags().StringVar(&createStatus, "status", "planning", "Epic status (planning, active, done)")
	createCmd.Flags().StringArrayVar(&createLabels, "label", nil, "Epic label (repeatable)")
	createCmd.Flags().BoolVar(&createEdit, "edit", false, "Write the epic in $EDITOR")
	createCmd.Flags().StringVar(&createCurrent, "current-state", "", "Current state section (markdown)")
	createCmd.Flags().StringVar(&createTarget, "target-state", "", "Target state section (markdown)")
	createCmd.Flags().StringVar(&createConstraint, "constraints", "", "Constraints section (markdown)")
	createCmd.Flags().StringVar(&createExclusion, "exclusions", "", "Exclusions section (markdown)")
	createCmd.Flags().StringVar(&createFreeform, "freeform", "", "Freeform fallback section (markdown); use - to read stdin")
}
