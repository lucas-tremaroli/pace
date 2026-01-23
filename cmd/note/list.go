package note

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var listEditor string
var listOutput string

type noteListResponse struct {
	Notes []note.NoteInfo `json:"notes"`
	Count int             `json:"count"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Browse and open your existing notes in a TUI",
	Long:  `Launch a TUI to browse, view, and open your existing notes. Use --json to output the list of notes in JSON format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := note.NewService()

		if cmd.Flags().Changed("json") {
			listOutput = "json"
		}

		if err != nil {
			if listOutput == "json" {
				output.Error(err)
			}
			return err
		}

		// JSON output mode
		if listOutput == "json" {
			notes, err := svc.ListNotes()
			if err != nil {
				output.Error(err)
			}
			output.JSON(noteListResponse{
				Notes: notes,
				Count: len(notes),
			})
			return nil
		}

		// TUI mode (default)
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
	listCmd.Flags().Bool("json", false, "Output result in JSON format")
}
