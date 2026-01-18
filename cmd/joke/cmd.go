package joke

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
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

		jokeText, err := svc.FetchJoke()
		if err != nil {
			return err
		}

		jokeStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			MarginTop(1)

		fmt.Println(jokeStyle.Render(jokeText))

		return nil
	},
}
