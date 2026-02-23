package note

import (
	"fmt"

	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var (
	readOutput string
	readAll    bool
)

type readAllResponse struct {
	Notes []note.Note `json:"notes"`
	Count int         `json:"count"`
}

var readCmd = &cobra.Command{
	Use:     "read [filename]",
	Aliases: []string{"cat"},
	Short:   "Read a note's content (alias: cat)",
	Long:    `Reads and outputs a note's content. Use --json for JSON format. Use --all to read all notes. Alias: 'pace note cat'`,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			readOutput = "json"
		}

		svc, err := note.NewService()
		if err != nil {
			if readOutput == "json" {
				output.Error(err)
			}
			return err
		}

		// Handle --all flag
		if readAll {
			notes, err := svc.ReadAllNotes()
			if err != nil {
				if readOutput == "json" {
					output.Error(err)
				}
				return fmt.Errorf("failed to read notes: %w", err)
			}

			if readOutput == "json" {
				output.JSON(readAllResponse{
					Notes: notes,
					Count: len(notes),
				})
				return nil
			}

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

		// Single note read (requires filename)
		if len(args) == 0 {
			return fmt.Errorf("filename required (or use --all to read all notes)")
		}

		filename := args[0]
		n, err := svc.ReadNoteWithMeta(filename)
		if err != nil {
			if readOutput == "json" {
				output.Error(err)
			}
			return fmt.Errorf("failed to read note: %w", err)
		}

		if readOutput == "json" {
			output.JSON(n)
			return nil
		}

		// Raw content output
		fmt.Print(n.Content)
		return nil
	},
}

func init() {
	readCmd.Flags().Bool("json", false, "Output in JSON format")
	readCmd.Flags().BoolVar(&readAll, "all", false, "Read all notes with full content")
}
