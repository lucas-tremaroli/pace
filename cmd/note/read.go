package note

import (
	"fmt"
	"path/filepath"

	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var readOutput string

var readCmd = &cobra.Command{
	Use:     "read <filename>",
	Aliases: []string{"cat"},
	Short:   "Read a note's content (alias: cat)",
	Long:    `Reads and outputs a note's content. Use --json for JSON format. Alias: 'pace note cat'`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

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

		content, err := svc.ReadNote(filename)
		if err != nil {
			if readOutput == "json" {
				output.Error(err)
			}
			return fmt.Errorf("failed to read note: %w", err)
		}

		if readOutput == "json" {
			path := svc.GetNotePath(filename)
			output.JSON(map[string]any{
				"filename": filepath.Base(path),
				"path":     path,
				"content":  content,
			})
			return nil
		}

		// Raw content output
		fmt.Print(content)
		return nil
	},
}

func init() {
	readCmd.Flags().Bool("json", false, "Output in JSON format")
}
