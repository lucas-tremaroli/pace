package note

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var content string
var editor string
var taskIDs []string

var createCmd = &cobra.Command{
	Use:     "create [filename]",
	GroupID: "manage",
	Short:   "Create a new note",
	Long:    `Creates a new markdown note with the specified filename and content.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithNoteService(func(svc *note.Service) error {
			var filename string
			if len(args) == 1 {
				filename = args[0]
			}

			if !cmd.Flags().Changed("content") {
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					stdinBytes, err := io.ReadAll(os.Stdin)
					if err != nil {
						return output.Error(err)
					}
					content = string(stdinBytes)
				}
			}

			if content != "" {
				if err := svc.WriteNote(filename, content); err != nil {
					return output.Error(err)
				}
				if cmd.Flags().Changed("editor") {
					return svc.OpenInEditor(filename, editor)
				}

				path := svc.GetNotePath(filename)

				if len(taskIDs) > 0 {
					if err := cmdutil.WithTaskService(func(taskSvc *task.Service) error {
						noteFilename := filepath.Base(path)
						if !strings.HasSuffix(noteFilename, ".md") {
							noteFilename += ".md"
						}
						for _, id := range taskIDs {
							if err := taskSvc.LinkNote(id, noteFilename); err != nil {
								return output.Error(err)
							}
						}
						return nil
					}); err != nil {
						return err
					}
				}

				output.Success("note created", map[string]string{
					"filename": filepath.Base(path),
					"path":     path,
				})
				return nil
			}

			if err := svc.WriteNote(filename, note.DefaultTemplate("Title")); err != nil {
				return output.Error(err)
			}
			return svc.OpenInEditor(filename, editor)
		})
	},
}

func init() {
	createCmd.Flags().StringVarP(&content, "content", "c", "", "Write content directly to the note without opening the editor")
	createCmd.Flags().StringVarP(&editor, "editor", "e", "nvim", "Editor to use for writing the note")
	createCmd.Flags().StringSliceVarP(&taskIDs, "task", "t", nil, "Task ID(s) to link to this note")
}
