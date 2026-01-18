package note

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/spf13/cobra"
)

var NoteCmd = &cobra.Command{
	Use:   "note [filename]",
	Short: "Opens a note in neovim",
	Long:  `Opens a markdown note in neovim. If no filename is provided, uses today's date (YYYY-MM-DD.md).`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine filename
		var filename string
		if len(args) == 1 {
			filename = args[0]
			if !strings.HasSuffix(filename, ".md") {
				filename += ".md"
			}
		} else {
			filename = time.Now().Format("2006-01-02") + ".md"
		}

		// Get notes directory
		paceDir, err := storage.GetpaceConfigDir()
		if err != nil {
			return fmt.Errorf("failed to get config directory: %w", err)
		}
		notesDir := filepath.Join(paceDir, "notes")

		// Ensure notes directory exists
		if err := os.MkdirAll(notesDir, 0755); err != nil {
			return fmt.Errorf("failed to create notes directory: %w", err)
		}

		// Open neovim with the file
		filePath := filepath.Join(notesDir, filename)
		nvim := exec.Command("nvim", filePath)
		nvim.Stdin = os.Stdin
		nvim.Stdout = os.Stdout
		nvim.Stderr = os.Stderr

		if err := nvim.Run(); err != nil {
			return fmt.Errorf("failed to open neovim: %w", err)
		}
		return nil
	},
}
