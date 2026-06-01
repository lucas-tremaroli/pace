package note

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/cmdutil"
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
	listPretty         bool
)


var listCmd = &cobra.Command{
	Use:     "list",
	GroupID: "manage",
	Short:   "List all notes",
	Long:    `List all notes in JSON format. Use --sort to change the order, --include-content to include full content, and --filter to filter by label.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithNoteService(func(svc *note.Service) error {
			notes, err := svc.ListNotesWithMeta(listIncludeContent)
			if err != nil {
				return output.Error(err)
			}

			if len(listFilters) > 0 {
				var filters []*note.NoteFilter
				for _, filterStr := range listFilters {
					f, err := note.ParseFilter(filterStr)
					if err != nil {
						return output.ErrorWithCode(err, output.ErrCodeInvalidParams, "Filter format: key=value (e.g. label=design)")
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

			if listHead > 0 && listHead < len(notes) {
				notes = notes[:listHead]
			}

			if listPretty {
				printNotesPretty(notes)
				return nil
			}

			if listFields != "" {
				maps, err := output.ToMapSlice(notes)
				if err != nil {
					return output.ErrorMsgWithCode(fmt.Sprintf("failed to filter fields: %v", err), output.ErrCodeInvalidParams, "")
				}
				fields := strings.Split(listFields, ",")
				output.JSON(map[string]any{
					"notes": output.FilterFields(maps, fields),
					"count": len(notes),
				})
				return nil
			}

			output.JSON(output.NoteListResponse{Notes: notes, Count: len(notes)})
			return nil
		})
	},
}

func init() {
	listCmd.Flags().StringVar(&listSort, "sort", "name", "Sort by: name, modified, created")
	listCmd.Flags().BoolVar(&listIncludeContent, "include-content", false, "Include full note content in output")
	listCmd.Flags().StringArrayVar(&listFilters, "filter", nil, "Filter notes (e.g., label=design). Can be repeated for AND semantics")
	listCmd.Flags().StringVar(&listFields, "fields", "", "Comma-separated fields to include. Available: filename, description, labels, modTime")
	listCmd.Flags().IntVar(&listHead, "head", 0, "Limit output to first N notes")
	listCmd.Flags().BoolVar(&listPretty, "pretty", false, "Human-readable formatted output")
}

func printNotesPretty(notes []note.Note) {
	fmt.Println()
	if len(notes) == 0 {
		fmt.Println(dimStyle.Render("No notes found."))
		fmt.Println()
		return
	}
	for _, n := range notes {
		name := strings.TrimSuffix(n.Filename, ".md")
		line := titleStyle.Render(name)
		if n.Description != "" {
			line += " " + dimStyle.Render("— "+n.Description)
		}
		if len(n.Labels) > 0 {
			line += " " + labelStyle.Render("["+strings.Join(n.Labels, ", ")+"]")
		}
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println(dimStyle.Render(fmt.Sprintf("%d note(s)", len(notes))))
}

func sortNotesWithMeta(notes []note.Note, sortBy string) {
	switch sortBy {
	case "modified", "created":
		slices.SortFunc(notes, func(a, b note.Note) int {
			return b.ModTime.Compare(a.ModTime)
		})
	default:
		slices.SortFunc(notes, func(a, b note.Note) int {
			return strings.Compare(a.Filename, b.Filename)
		})
	}
}
