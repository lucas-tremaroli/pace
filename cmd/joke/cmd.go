package joke

import (
	"context"
	"fmt"
	"strings"
	"time"

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

		jokeText, err := svc.FetchJoke(context.Background())
		if err != nil {
			return err
		}

		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#45B7D1")).Bold(true)

		// Split into setup/punchline if the joke contains a question mark
		if idx := strings.Index(jokeText, "?"); idx != -1 {
			setup := jokeText[:idx+1]
			punchline := strings.TrimSpace(jokeText[idx+1:])

			typewrite(setup, style)
			fmt.Println()

			if punchline != "" {
				time.Sleep(1500 * time.Millisecond)
				typewrite(punchline, style)
				fmt.Println()
			}
		} else {
			typewrite(jokeText, style)
			fmt.Println()
		}

		return nil
	},
}

func typewrite(text string, style lipgloss.Style) {
	for _, ch := range text {
		fmt.Print(style.Render(string(ch)))
		time.Sleep(30 * time.Millisecond)
	}
}

func init() {
	JokeCmd.GroupID = "recharge"
}
