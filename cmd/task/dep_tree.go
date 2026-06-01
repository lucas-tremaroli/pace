package task

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var (
	treeDirection string
	treeStatus    string
	treeMaxDepth  int
)

var depTreeCmd = &cobra.Command{
	Use:   "tree <task-id>",
	Short: "Visualize dependency tree for a task",
	Long: `Shows an ASCII tree of blockers (what blocks this task) and what this task blocks.

Use --direction to control which relationships to show:
  - up:   Show what blocks this task (default)
  - down: Show what this task blocks
  - both: Show full graph in both directions

Examples:
  pace task dep tree pace-abc                      # Show what blocks pace-abc
  pace task dep tree pace-abc --direction=down     # Show what pace-abc blocks
  pace task dep tree pace-abc --status=todo        # Only show todo tasks
  pace task dep tree pace-abc -d 2                 # Limit to 2 levels deep`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		if treeDirection != "down" && treeDirection != "up" && treeDirection != "both" {
			return output.ErrorMsgWithCode(
				fmt.Sprintf("invalid direction: %s (valid: down, up, both)", treeDirection),
				output.ErrCodeInvalidParams,
				"Valid values: up, down, both",
			)
		}

		var filterStatus *task.Status
		if treeStatus != "" {
			s, err := task.ParseStatus(treeStatus)
			if err != nil {
				return output.ErrorWithCode(err, output.ErrCodeInvalidStatus, "Valid values: todo, in-progress, done")
			}
			filterStatus = &s
		}

		svc, err := task.NewService()
		if err != nil {
			return output.Error(err)
		}
		defer svc.Close()

		tasks, err := svc.LoadAllTasks()
		if err != nil {
			return output.Error(err)
		}

		taskMap := make(map[string]task.Task)
		for _, t := range tasks {
			taskMap[t.ID()] = t
		}

		rootTask, exists := taskMap[taskID]
		if !exists {
			return output.ErrorMsgWithCode("task not found: "+taskID, output.ErrCodeTaskNotFound, "Use pace task list to see available task IDs")
		}

		opts := treeOptions{
			direction:    treeDirection,
			filterStatus: filterStatus,
			maxDepth:     treeMaxDepth,
			taskMap:      taskMap,
		}
		printDepTree(rootTask, opts)
		return nil
	},
}

var (
	treeRootStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	treeBlockerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	treeBlocksStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	treeNodeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	treeBranchStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	treeLabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	treeReadyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
)

type treeOptions struct {
	direction    string
	filterStatus *task.Status
	maxDepth     int
	taskMap      map[string]task.Task
}

func printDepTree(root task.Task, opts treeOptions) {
	fmt.Println()

	showBlockers := opts.direction == "up" || opts.direction == "both"
	showBlocks := opts.direction == "down" || opts.direction == "both"

	blockers := root.BlockedBy()
	if showBlockers && len(blockers) > 0 {
		fmt.Println(treeBlockerStyle.Render("BLOCKED BY:"))
		printTree(blockers, opts, "", make(map[string]bool), true, 0)
		fmt.Println()
	}

	fmt.Print(treeRootStyle.Render(fmt.Sprintf("► %s: %s", root.ID(), root.Title())))
	if isTaskReady(root, opts.taskMap) {
		fmt.Print(" " + treeReadyStyle.Render("[READY]"))
	}
	fmt.Println()
	printTaskStatus(root)
	fmt.Println()

	blocks := root.Blocks()
	if showBlocks && len(blocks) > 0 {
		fmt.Println(treeBlocksStyle.Render("BLOCKS:"))
		printTree(blocks, opts, "", make(map[string]bool), false, 0)
		fmt.Println()
	}

	hasBlockers := showBlockers && len(blockers) > 0
	hasBlocks := showBlocks && len(blocks) > 0
	if !hasBlockers && !hasBlocks {
		fmt.Println(treeLabelStyle.Render("No dependencies found."))
		fmt.Println()
	}
}

func isTaskReady(t task.Task, taskMap map[string]task.Task) bool {
	if t.Status() == task.Done {
		return false
	}
	for _, blockerID := range t.BlockedBy() {
		if blocker, exists := taskMap[blockerID]; exists {
			if blocker.Status() != task.Done {
				return false
			}
		}
	}
	return true
}

func printTaskStatus(t task.Task) {
	var statusStr string
	switch t.Status() {
	case task.Todo:
		statusStr = todoStyle.Render("○ todo")
	case task.InProgress:
		statusStr = progressStyle.Render("● in-progress")
	case task.Done:
		statusStr = doneStyle.Render("● done")
	}
	fmt.Printf("  %s", statusStr)

	if p := t.Priority(); p > 0 {
		pStyle, ok := priorityStyles[p]
		if !ok {
			pStyle = priorityStyle
		}
		fmt.Printf(" %s", pStyle.Render(task.PriorityName(p)))
	}
	fmt.Println()
}

func printTree(taskIDs []string, opts treeOptions, prefix string, visited map[string]bool, isBlocker bool, depth int) {
	var filteredIDs []string
	for _, id := range taskIDs {
		if t, exists := opts.taskMap[id]; exists {
			if opts.filterStatus == nil || t.Status() == *opts.filterStatus {
				filteredIDs = append(filteredIDs, id)
			}
		} else {
			filteredIDs = append(filteredIDs, id)
		}
	}

	for i, id := range filteredIDs {
		isLast := i == len(filteredIDs)-1
		printTreeNode(id, opts, prefix, isLast, visited, isBlocker, depth)
	}
}

func printTreeNode(id string, opts treeOptions, prefix string, isLast bool, visited map[string]bool, isBlocker bool, depth int) {
	if depth >= opts.maxDepth {
		return
	}

	var branch, childPrefix string
	if isLast {
		branch = "└── "
		childPrefix = prefix + "    "
	} else {
		branch = "├── "
		childPrefix = prefix + "│   "
	}

	t, exists := opts.taskMap[id]
	if !exists {
		fmt.Printf("%s%s%s\n",
			treeBranchStyle.Render(prefix+branch),
			treeLabelStyle.Render(id),
			treeLabelStyle.Render(" (not found)"))
		return
	}

	if visited[id] {
		fmt.Printf("%s%s%s\n",
			treeBranchStyle.Render(prefix+branch),
			treeNodeStyle.Render(fmt.Sprintf("%s: %s", id, t.Title())),
			treeLabelStyle.Render(" (cycle)"))
		return
	}

	var statusIndicator string
	switch t.Status() {
	case task.Todo:
		statusIndicator = todoStyle.Render("○")
	case task.InProgress:
		statusIndicator = progressStyle.Render("●")
	case task.Done:
		statusIndicator = doneStyle.Render("●")
	}

	nodeLine := fmt.Sprintf("%s: %s", id, truncateTitle(t.Title(), 50))

	if isTaskReady(t, opts.taskMap) {
		nodeLine += " " + treeReadyStyle.Render("[READY]")
	}

	fmt.Printf("%s%s %s\n",
		treeBranchStyle.Render(prefix+branch),
		statusIndicator,
		treeNodeStyle.Render(nodeLine))

	visited[id] = true

	var children []string
	if isBlocker {
		children = t.BlockedBy()
	} else {
		children = t.Blocks()
	}

	if len(children) > 0 {
		printTree(children, opts, childPrefix, visited, isBlocker, depth+1)
	}

	delete(visited, id)
}

func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return strings.TrimSpace(title[:maxLen-3]) + "..."
}
