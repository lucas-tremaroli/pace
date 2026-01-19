package note

import (
	"fmt"
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

		for {
			picker := note.NewPicker(svc)
			p := tea.NewProgram(picker, tea.WithAltScreen())

			finalModel, err := p.Run()
			if err != nil {
				return err
			}

			m, ok := finalModel.(note.Picker)
			if !ok || !m.ShouldOpenFile() {
				break
			}

			// Clear screen to avoid flash between picker and editor
			fmt.Print("\033[H\033[2J")

			fileToOpen := strings.TrimSuffix(m.FileToOpen(), ".md")
			if err := svc.OpenInEditor(fileToOpen); err != nil {
				return err
			}
		}

		return nil
	},
}
