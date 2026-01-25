package note

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var content string
var editor string
var createOutput string

var createCmd = &cobra.Command{
	Use:   "create [filename]",
	Short: "Create a new note",
	Long:  `Creates a new markdown note with the specified filename and content. Use --json for JSON output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := note.NewService()

		if cmd.Flags().Changed("json") {
			createOutput = "json"
		}

		if err != nil {
			if createOutput == "json" {
				output.Error(err)
			}
			return err
		}

		var filename string

		if len(args) == 1 {
			filename = args[0]
		}

		// Read from stdin if piped and no content flag provided
		if !cmd.Flags().Changed("content") {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				stdinBytes, err := io.ReadAll(os.Stdin)
				if err != nil {
					if createOutput == "json" {
						output.Error(err)
					}
					return err
				}
				content = string(stdinBytes)
			}
		}

		if content != "" {
			if err := svc.WriteNote(filename, content); err != nil {
				if createOutput == "json" {
					output.Error(err)
				}
				return err
			}
			if cmd.Flags().Changed("editor") {
				return svc.OpenInEditor(filename, editor)
			}

			path := svc.GetNotePath(filename)

			if createOutput == "json" {
				output.Success("note created", map[string]string{
					"filename": filepath.Base(path),
					"path":     path,
				})
				return nil
			}

			successStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("10"))
			pathStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("12")).
				Underline(true)
			fmt.Println(successStyle.Render("✓ Note created: ") + pathStyle.Render(path))
			return nil
		}
		return svc.OpenInEditor(filename, editor)
	},
}

func init() {
	createCmd.Flags().StringVarP(&content, "content", "c", "", "Write content directly to the note without opening the editor")
	createCmd.Flags().StringVarP(&editor, "editor", "e", "nvim", "Editor to use for writing the note")
	createCmd.Flags().Bool("json", false, "Output result in JSON format")
}
