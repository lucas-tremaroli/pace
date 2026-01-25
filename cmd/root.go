package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/cmd/config"
	"github.com/lucas-tremaroli/pace/cmd/joke"
	"github.com/lucas-tremaroli/pace/cmd/note"
	"github.com/lucas-tremaroli/pace/cmd/task"
	"github.com/lucas-tremaroli/pace/cmd/tick"
	"github.com/spf13/cobra"
)

var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	commandStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
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

func styledHelp(cmd *cobra.Command, _ []string) {
	var b strings.Builder

	// Description
	if cmd.Long != "" {
		b.WriteString(cmd.Long + "\n\n")
	}

	// Usage
	b.WriteString(headerStyle.Render("Usage") + "\n")
	b.WriteString("  " + dimStyle.Render(cmd.UseLine()) + "\n")
	if cmd.HasAvailableSubCommands() {
		b.WriteString("  " + dimStyle.Render(cmd.CommandPath()+" [command]") + "\n")
	}
	b.WriteString("\n")

	// Command groups
	groups := cmd.Groups()
	for _, group := range groups {
		b.WriteString(headerStyle.Render(group.Title) + "\n")
		for _, sub := range cmd.Commands() {
			if sub.GroupID == group.ID && sub.IsAvailableCommand() {
				name := commandStyle.Render(fmt.Sprintf("  %-12s", sub.Name()))
				b.WriteString(name + sub.Short + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Additional commands (ungrouped)
	var additional []*cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.GroupID == "" && sub.IsAvailableCommand() {
			additional = append(additional, sub)
		}
	}
	if len(additional) > 0 {
		b.WriteString(headerStyle.Render("Additional Commands") + "\n")
		for _, sub := range additional {
			name := commandStyle.Render(fmt.Sprintf("  %-12s", sub.Name()))
			b.WriteString(name + sub.Short + "\n")
		}
		b.WriteString("\n")
	}

	// Flags
	if cmd.HasAvailableLocalFlags() {
		b.WriteString(headerStyle.Render("Flags") + "\n")
		b.WriteString(cmd.LocalFlags().FlagUsages())
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(dimStyle.Render(fmt.Sprintf("Use \"%s [command] --help\" for more information about a command.", cmd.CommandPath())))
	b.WriteString("\n")

	fmt.Print(b.String())
}

func init() {
	rootCmd.AddGroup(&cobra.Group{ID: "core", Title: "Core"})
	rootCmd.AddGroup(&cobra.Group{ID: "configuration", Title: "Configuration"})
	rootCmd.AddGroup(&cobra.Group{ID: "recharge", Title: "Recharge"})

	rootCmd.AddCommand(task.TaskCmd)
	rootCmd.AddCommand(note.NoteCmd)
	rootCmd.AddCommand(tick.TickCmd)
	rootCmd.AddCommand(joke.JokeCmd)
	rootCmd.AddCommand(config.ConfigCmd)

	rootCmd.SetHelpFunc(styledHelp)
}
