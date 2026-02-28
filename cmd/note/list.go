package note

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/spf13/cobra"
)

var (
	listSort           string
	listIncludeContent bool
	listFilters        []string
	listFields         string
	listHead           int
)

type noteListResponse struct {
	Notes []note.Note `json:"notes"`
	Count int         `json:"count"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notes",
	Long:  `List all notes in JSON format. Use --sort to change the order, --include-content to include full content, and --filter to filter by label.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := note.NewService()
		if err != nil {
			output.Error(err)
			return nil
		}

		notes, err := svc.ListNotesWithMeta(listIncludeContent)
		if err != nil {
			output.Error(err)
			return nil
		}

		// Parse and apply filters
		if len(listFilters) > 0 {
			var filters []*note.NoteFilter
			for _, filterStr := range listFilters {
				f, err := note.ParseFilter(filterStr)
				if err != nil {
					output.Error(err)
					return nil
				}
				filters = append(filters, f)
			}

			merged := note.MergeFilters(filters)
			var filtered []note.Note
			for _, n := range notes {
				if merged.Matches(n) {
					filtered = append(filtered, n)
				}
			}
			notes = filtered
		}

		sortNotesWithMeta(notes, listSort)

		// Apply head truncation
		if listHead > 0 && listHead < len(notes) {
			notes = notes[:listHead]
		}

		if listFields != "" {
			maps, err := output.ToMapSlice(notes)
			if err != nil {
				output.ErrorMsg(fmt.Sprintf("failed to filter fields: %v", err))
				return nil
			}
			fields := strings.Split(listFields, ",")
			output.JSON(map[string]any{
				"notes": output.FilterFields(maps, fields),
				"count": len(notes),
			})
			return nil
		}

		output.JSON(noteListResponse{
			Notes: notes,
			Count: len(notes),
		})
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listSort, "sort", "name", "Sort by: name, modified, created")
	listCmd.Flags().BoolVar(&listIncludeContent, "include-content", false, "Include full note content in output")
	listCmd.Flags().StringArrayVar(&listFilters, "filter", nil, "Filter notes (e.g., label=design). Can be repeated for AND semantics")
	listCmd.Flags().StringVar(&listFields, "fields", "", "Comma-separated list of fields to include (e.g. filename,description)")
	listCmd.Flags().IntVar(&listHead, "head", 0, "Limit output to first N notes")
}

func sortNotesWithMeta(notes []note.Note, sortBy string) {
	switch sortBy {
	case "modified", "created":
		// Sort by modification time (newest first)
		slices.SortFunc(notes, func(a, b note.Note) int {
			return b.ModTime.Compare(a.ModTime)
		})
	default: // "name"
		// Sort alphabetically by filename
		slices.SortFunc(notes, func(a, b note.Note) int {
			return strings.Compare(a.Filename, b.Filename)
		})
	}
}
