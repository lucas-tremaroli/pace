package cmd

import (
	"fmt"
	"os"

	"github.com/lucas-tremaroli/pace/cmd/joke"
	"github.com/lucas-tremaroli/pace/cmd/note"
	"github.com/lucas-tremaroli/pace/cmd/task"
	"github.com/lucas-tremaroli/pace/cmd/tick"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:  "pace",
	Long: `A simple CLI tool to manage tasks, notes, and more.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func SetVersionInfo(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
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
	rootCmd.AddCommand(tick.TickCmd)
	rootCmd.AddCommand(joke.JokeCmd)
}
