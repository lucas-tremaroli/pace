package joke

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/lucas-tremaroli/pace/internal/joke"
	"github.com/spf13/cobra"
)

var JokeCmd = &cobra.Command{
	Use:   "joke",
	Short: "Displays a random dad joke",
	Long:  `Fetches a random dad joke from icanhazdadjoke.com just 4 fun.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := joke.NewService()

		jokeText, err := svc.FetchJoke(context.Background())
		if err != nil {
			return err
		}

		// Get terminal width, default to 80 if unavailable
		width := 80
		if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
			width = w
		}

		// Account for border (2) + padding (4) + margin
		maxWidth := width - 10
		if maxWidth < 20 {
			maxWidth = 20
		}

		jokeStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			MarginTop(1).
			Width(maxWidth)

		fmt.Println(jokeStyle.Render(jokeText))

		return nil
	},
}
