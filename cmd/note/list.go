package note

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/spf13/cobra"
)

var listEditor string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Browse and open your existing notes in a TUI",
	Long:  `Launch a TUI to browse, view, and open your existing notes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := note.NewService()
		if err != nil {
			return err
		}

		for {
			// Clear screen to avoid flash between transitions
			fmt.Print("\033[H\033[2J")

			picker := note.NewPicker(svc)
			p := tea.NewProgram(picker, tea.WithAltScreen())

			finalModel, err := p.Run()
			if err != nil {
				return err
			}

			m, ok := finalModel.(note.Picker)
			if !ok {
				break
			}

			fileToOpen := m.FileToOpen()
			if fileToOpen == "" {
				break
			}

			filename := strings.TrimSuffix(fileToOpen, ".md")

			if m.ShouldViewFile() {
				content, err := svc.ReadNote(filename)
				if err != nil {
					return fmt.Errorf("failed to read note: %w", err)
				}

				rendered := note.RenderMarkdown(content)
				viewer := note.NewViewer(fileToOpen, rendered)
				vp := tea.NewProgram(viewer, tea.WithAltScreen())
				if _, err := vp.Run(); err != nil {
					return err
				}
				continue
			}

			if m.ShouldOpenFile() {
				if err := svc.OpenInEditor(filename, listEditor); err != nil {
					return err
				}
				continue
			}

			break
		}

		return nil
	},
}

func init() {
	listCmd.Flags().StringVarP(&listEditor, "editor", "e", "nvim", "Editor to use for opening notes")
}
