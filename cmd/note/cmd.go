package note

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/spf13/cobra"
)

var content string

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

		if content != "" {
			if err := svc.WriteNote(filename, content); err != nil {
				return err
			}
			successStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("10"))
			pathStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("12")).
				Underline(true)
			fmt.Println(successStyle.Render("✓ Note created: ") + pathStyle.Render(svc.GetNotePath(filename)))
			return nil
		}
		return svc.OpenInEditor(filename)
	},
}

func init() {
	NoteCmd.Flags().StringVarP(&content, "content", "c", "", "Write content directly to the note without opening the editor")
}
