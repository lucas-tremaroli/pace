package note

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/spf13/cobra"
)

var (
	content  string
	listFlag bool
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

		if listFlag {
			return runFilePicker(svc)
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

func runFilePicker(svc *note.Service) error {
	picker := note.NewPicker(svc)
	p := tea.NewProgram(picker, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if we should open a file after exiting the TUI
	if m, ok := finalModel.(note.Picker); ok {
		if m.ShouldOpenFile() {
			fileToOpen := m.FileToOpen()
			// Strip .md extension for OpenInEditor since it adds it back
			fileToOpen = strings.TrimSuffix(fileToOpen, ".md")
			return svc.OpenInEditor(fileToOpen)
		}
	}

	return nil
}

func init() {
	NoteCmd.Flags().StringVarP(&content, "content", "c", "", "Write content directly to the note without opening the editor")
	NoteCmd.Flags().BoolVarP(&listFlag, "list", "l", false, "List and browse existing notes")
}
