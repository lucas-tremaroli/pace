package note

import (
	"errors"
	"fmt"
	"os"

	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var (
	readRaw bool
	readAll bool
)

type readAllResponse struct {
	Notes []note.Note `json:"notes"`
	Count int         `json:"count"`
}

var readCmd = &cobra.Command{
	Use:     "read [filename]",
	GroupID: "manage",
	Aliases: []string{"cat"},
	Short:   "Read a note's content (alias: cat)",
	Long:    `Reads and outputs a note's content. Use --raw for raw markdown output. Use --all to read all notes. Alias: 'pace note cat'`,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := note.NewService()
		if err != nil {
			output.Error(err)
		}

		// Handle --all flag
		if readAll {
			notes, err := svc.ReadAllNotes()
			if err != nil {
				output.Error(err)
			}

			if readRaw {
				// Raw output for all notes
				for i, n := range notes {
					if i > 0 {
						fmt.Print("\n---\n\n")
					}
					fmt.Printf("# %s\n\n", n.Filename)
					fmt.Print(n.Content)
				}
				return nil
			}

			output.JSON(readAllResponse{
				Notes: notes,
				Count: len(notes),
			})
			return nil
		}

		// Single note read (requires filename)
		if len(args) == 0 {
			output.ErrorMsgWithCode("filename required (or use --all to read all notes)", output.ErrCodeMissingField, "")
		}

		filename := args[0]
		n, err := svc.ReadNoteWithMeta(filename)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				output.ErrorMsgWithCode("note not found: "+filename, output.ErrCodeNoteNotFound, "Use pace note list to see available notes")
			}
			output.Error(err)
		}

		if readRaw {
			// Raw content output
			fmt.Print(n.Content)
			return nil
		}

		output.JSON(n)
		return nil
	},
}

func init() {
	readCmd.Flags().BoolVar(&readRaw, "raw", false, "Output raw markdown content without JSON wrapper")
	readCmd.Flags().BoolVar(&readAll, "all", false, "Read all notes with full content")
}
