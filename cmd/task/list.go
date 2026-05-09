package task

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/cmdutil"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	idStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	priorityStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("202")).Bold(true)
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	noteStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	depStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	labelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	todoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	progressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	doneStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	blockedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	countStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	p1Style = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	p2Style = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	p3Style = lipgloss.NewStyle().Foreground(lipgloss.Color("146"))

	labelTaskStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	labelBugStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("167"))
	labelFeatureStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	labelDocsStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
)

var (
	listPretty         bool
	listFilterStatus   string
	listFilterPriority []string
	listFilterLabel    string
	listFields         string
	listHead           int
	listReady          bool
)


var listCmd = &cobra.Command{
	Use:     "list",
	GroupID: "manage",
	Short:   "List all tasks",
	Long:    `Outputs all tasks. Use --pretty for human-readable format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmdutil.WithTaskService(func(svc *task.Service) error {
			tasks, err := svc.LoadAllTasks()
			if err != nil {
				return output.Error(err)
			}

			if listPretty && listFields != "" {
				return output.ErrorMsg("--pretty and --fields cannot be used together")
			}

			filter := task.TaskFilter{}
			if listFilterLabel != "" {
				filter.Label = &listFilterLabel
			}
			if listFilterStatus != "" {
				status, err := task.ParseStatus(listFilterStatus)
				if err != nil {
					return output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: todo, in-progress, done")
				}
				filter.Status = &status
			}
			for _, p := range listFilterPriority {
				pri, err := task.ParsePriority(p)
				if err != nil {
					return output.ErrorMsgWithCode(
						err.Error(),
						output.ErrCodeInvalidPriority,
						task.ValidPriorityHelp,
					)
				}
				filter.Priorities = append(filter.Priorities, pri)
			}
			filter.Ready = listReady
			tasks = filter.Apply(tasks)

			slices.SortFunc(tasks, func(a, b task.Task) int {
				return a.Priority() - b.Priority()
			})

			if listHead > 0 && listHead < len(tasks) {
				tasks = tasks[:listHead]
			}

			if listPretty {
				printTasksPretty(tasks)
				return nil
			}

			taskJSONs := make([]task.TaskJSON, len(tasks))
			for i, t := range tasks {
				taskJSONs[i] = t.ToJSON()
			}

			if listFields != "" {
				maps, err := output.ToMapSlice(taskJSONs)
				if err != nil {
					return output.ErrorMsg(fmt.Sprintf("failed to filter fields: %v", err))
				}
				fields := strings.Split(listFields, ",")
				output.JSON(map[string]any{
					"tasks": output.FilterFields(maps, fields),
					"count": len(taskJSONs),
				})
				return nil
			}

			output.JSON(output.TaskListResponse{Tasks: taskJSONs, Count: len(taskJSONs)})
			return nil
		})
	},
}

func init() {
	listCmd.Flags().BoolVar(&listPretty, "pretty", false, "Human-readable formatted output")
	listCmd.Flags().StringVar(&listFilterStatus, "status", "", "Filter by status (todo, in-progress, done)")
	listCmd.Flags().StringArrayVar(&listFilterPriority, "priority", nil, "Filter by priority (high, medium, low or 1-3, repeatable)")
	listCmd.Flags().StringVar(&listFilterLabel, "label", "", "Filter by label")
	listCmd.Flags().StringVar(&listFields, "fields", "", "Comma-separated fields to include. Available: id, title, description, status, priority, blocked_by, blocks, label, notes, link")
	listCmd.Flags().IntVar(&listHead, "head", 0, "Limit output to first N tasks")
	listCmd.Flags().BoolVar(&listReady, "ready", false, "Only show tasks ready to work on (not blocked, not done)")
}

func printTasksPretty(tasks []task.Task) {
	if len(tasks) == 0 {
		fmt.Println(countStyle.Render("No tasks found."))
		return
	}

	fmt.Println()
	for _, t := range tasks {
		fmt.Println(formatTaskPretty(t))
	}
	fmt.Println()
	fmt.Println(countStyle.Render(fmt.Sprintf("%d task(s) \n", len(tasks))))
	printLegend()
}

func printLegend() {
	status := countStyle.Render("Status: ") +
		todoStyle.Render("○") + countStyle.Render(" todo  ") +
		progressStyle.Render("●") + countStyle.Render(" in-progress  ") +
		doneStyle.Render("●") + countStyle.Render(" done  ") +
		blockedStyle.Render("⊘") + countStyle.Render(" blocked")
	fmt.Println(status)
}

func formatTaskPretty(t task.Task) string {
	var parts []string

	isBlocked := len(t.BlockedBy()) > 0
	if isBlocked {
		parts = append(parts, blockedStyle.Render("⊘"))
	} else {
		switch t.Status() {
		case task.Todo:
			parts = append(parts, todoStyle.Render("○"))
		case task.InProgress:
			parts = append(parts, progressStyle.Render("●"))
		case task.Done:
			parts = append(parts, doneStyle.Render("●"))
		}
	}

	parts = append(parts, idStyle.Render(t.ID()))
	parts = append(parts, titleStyle.Render(t.Title()))

	if label := t.Label(); label != "" {
		var lStyle lipgloss.Style
		switch label {
		case "bug":
			lStyle = labelBugStyle
		case "feature":
			lStyle = labelFeatureStyle
		case "docs":
			lStyle = labelDocsStyle
		default:
			lStyle = labelTaskStyle
		}
		parts = append(parts, lStyle.Render(fmt.Sprintf("[%s]", label)))
	}

	if p := t.Priority(); p > 0 {
		var pStyle lipgloss.Style
		switch p {
		case 1:
			pStyle = p1Style
		case 2:
			pStyle = p2Style
		case 3:
			pStyle = p3Style
		default:
			pStyle = priorityStyle
		}
		parts = append(parts, pStyle.Render(task.PriorityName(p)))
	}

	if len(t.Notes()) > 0 {
		parts = append(parts, noteStyle.Render(fmt.Sprintf("(notes:%d)", len(t.Notes()))))
	}
	if len(t.Blocks()) > 0 {
		parts = append(parts, depStyle.Render(fmt.Sprintf("(blocks:%d)", len(t.Blocks()))))
	}

	return strings.Join(parts, " ")
}
