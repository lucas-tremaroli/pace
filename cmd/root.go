package cmd

import (
	"fmt"
	"os"

	"github.com/lucas-tremaroli/pace/cmd/focus"
	"github.com/lucas-tremaroli/pace/cmd/note"
	"github.com/lucas-tremaroli/pace/cmd/task"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:  "pace",
	Long: `A simple CLI tool to manage tasks, notes, and more.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(task.TaskCmd)
	rootCmd.AddCommand(note.NoteCmd)
	rootCmd.AddCommand(focus.FocusCmd)
}
