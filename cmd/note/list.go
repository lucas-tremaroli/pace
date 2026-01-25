package note

import (
	"slices"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var listSort string

type noteListResponse struct {
	Notes []note.NoteInfo `json:"notes"`
	Count int             `json:"count"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notes",
	Long:  `List all notes in JSON format. Use --sort to change the order.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := note.NewService()
		if err != nil {
			output.Error(err)
			return nil
		}

		notes, err := svc.ListNotes()
		if err != nil {
			output.Error(err)
			return nil
		}

		sortNotes(notes, listSort)
		output.JSON(noteListResponse{
			Notes: notes,
			Count: len(notes),
		})
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listSort, "sort", "name", "Sort by: name, modified, created")
}

func sortNotes(notes []note.NoteInfo, sortBy string) {
	switch sortBy {
	case "modified", "created":
		// Sort by modification time (newest first)
		slices.SortFunc(notes, func(a, b note.NoteInfo) int {
			return b.ModTime.Compare(a.ModTime)
		})
	default: // "name"
		// Sort alphabetically by filename
		slices.SortFunc(notes, func(a, b note.NoteInfo) int {
			return strings.Compare(a.Filename, b.Filename)
		})
	}
}
