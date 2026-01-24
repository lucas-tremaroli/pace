package task

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	idStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	typeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	priorityStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("202")).Bold(true)
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	labelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	depStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	todoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	progressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	doneStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	blockedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	countStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	// Priority styles
	p1Style = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	p2Style = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	p3Style = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	p4Style = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

var listPretty bool

type taskListResponse struct {
	Tasks []task.TaskJSON `json:"tasks"`
	Count int             `json:"count"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Long:  `Outputs all tasks. Use --pretty for human-readable format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		tasks, err := svc.LoadAllTasks()
		if err != nil {
			output.Error(err)
		}

		// Sort by priority (P1 first, P4 last)
		slices.SortFunc(tasks, func(a, b task.Task) int {
			return a.Priority() - b.Priority()
		})

		if listPretty {
			printTasksPretty(tasks)
			return nil
		}

		taskJSONs := make([]task.TaskJSON, len(tasks))
		for i, t := range tasks {
			taskJSONs[i] = t.ToJSON()
		}

		output.JSON(taskListResponse{
			Tasks: taskJSONs,
			Count: len(taskJSONs),
		})
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listPretty, "pretty", false, "Human-readable formatted output")
}

// printTasksPretty prints tasks in a human-readable format
func printTasksPretty(tasks []task.Task) {
	if len(tasks) == 0 {
		fmt.Println(countStyle.Render("No tasks found."))
		return
	}

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

	priority := countStyle.Render("Priority: ") +
		p1Style.Render("P1") + countStyle.Render(" urgent  ") +
		p2Style.Render("P2") + countStyle.Render(" high  ") +
		p3Style.Render("P3") + countStyle.Render(" normal  ") +
		p4Style.Render("P4") + countStyle.Render(" low")
	fmt.Println(priority)
}

// formatTaskPretty formats a single task for pretty printing
func formatTaskPretty(t task.Task) string {
	var parts []string

	// Check if blocked
	isBlocked := len(t.BlockedBy()) > 0

	// Status symbol (blocked overrides other statuses visually)
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

	// ID
	parts = append(parts, idStyle.Render(t.ID()))

	// Type symbol
	parts = append(parts, typeStyle.Render(fmt.Sprintf("[%s]", t.Type().Symbol())))

	// Priority with color coding
	if p := t.Priority(); p > 0 {
		var pStyle lipgloss.Style
		switch p {
		case 1:
			pStyle = p1Style
		case 2:
			pStyle = p2Style
		case 3:
			pStyle = p3Style
		case 4:
			pStyle = p4Style
		default:
			pStyle = priorityStyle
		}
		parts = append(parts, pStyle.Render(fmt.Sprintf("P%d", p)))
	}

	// Title
	parts = append(parts, titleStyle.Render(t.Title()))

	// Labels
	for _, label := range t.Labels() {
		parts = append(parts, labelStyle.Render(fmt.Sprintf("[%s]", label)))
	}

	// Dependency indicators
	if isBlocked {
		parts = append(parts, depStyle.Render(fmt.Sprintf("(blocked:%d)", len(t.BlockedBy()))))
	}
	if len(t.Blocks()) > 0 {
		parts = append(parts, depStyle.Render(fmt.Sprintf("(blocks:%d)", len(t.Blocks()))))
	}

	return strings.Join(parts, " ")
}
