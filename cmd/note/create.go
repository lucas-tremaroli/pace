package note

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/spf13/cobra"
)

var content string
var editor string

var createCmd = &cobra.Command{
	Use:   "create [filename]",
	Short: "Create a new note",
	Long:  `Creates a new markdown note with the specified filename and content.`,
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
			if cmd.Flags().Changed("editor") {
				return svc.OpenInEditor(filename, editor)
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
		return svc.OpenInEditor(filename, editor)
	},
}

func init() {
	createCmd.Flags().StringVarP(&content, "content", "c", "", "Write content directly to the note without opening the editor")
	createCmd.Flags().StringVarP(&editor, "editor", "e", "nvim", "Editor to use for writing the note")
}
