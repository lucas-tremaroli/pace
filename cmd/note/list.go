package note

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Browse and open your existing notes in a TUI",
	Long:  `Launch a TUI to browse and open your existing notes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := note.NewService()
		if err != nil {
			return err
		}

		picker := note.NewPicker(svc)
		p := tea.NewProgram(picker, tea.WithAltScreen())

		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		if m, ok := finalModel.(note.Picker); ok {
			if m.ShouldOpenFile() {
				fileToOpen := m.FileToOpen()
				fileToOpen = strings.TrimSuffix(fileToOpen, ".md")
				return svc.OpenInEditor(fileToOpen)
			}
		}

		return nil
	},
}
