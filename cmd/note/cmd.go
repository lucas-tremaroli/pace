package note

import (
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/spf13/cobra"
)

var NoteCmd = &cobra.Command{
	Use:   "note [filename]",
	Short: "Opens a note in neovim",
	Long:  `Opens a markdown note in neovim. If no filename is provided, uses today's date (YYYY-MM-DD.md).`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := note.NewService()
		if err != nil {
			return err
		}

		var filename string
		if len(args) == 1 {
			filename = args[0]
		}
		return svc.OpenInEditor(filename)
	},
}
