package note

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/spf13/cobra"
)

var NoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Opens the note management tool",
	Long:  `Opens the note management tool`,
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(note.NewNoteEditor())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running program: %v", err)
			os.Exit(1)
		}
	},
}
