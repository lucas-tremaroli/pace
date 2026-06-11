package tuiproto

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/tuiproto"
	"github.com/spf13/cobra"
)

var TuiProtoCmd = &cobra.Command{
	Use:    "tui-proto",
	Short:  "Prototype tabbed TUI (kanban + notes)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := tuiproto.New()
		if err != nil {
			return fmt.Errorf("init tui-proto: %w", err)
		}
		if _, err := tea.NewProgram(r, tea.WithAltScreen()).Run(); err != nil {
			return fmt.Errorf("run tui-proto: %w", err)
		}
		return nil
	},
}
