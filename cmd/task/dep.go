package task

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/task"
	"github.com/spf13/cobra"
)

var depCmd = &cobra.Command{
	Use:   "dep",
	Short: "Manage task dependencies",
	Long:  `Manage blocking relationships between tasks.`,
}

var depAddCmd = &cobra.Command{
	Use:   "add <blocker-id> <blocked-id>",
	Short: "Add a dependency (blocker blocks blocked)",
	Long:  `Creates a blocking relationship where the first task blocks the second task.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		blockerID := args[0]
		blockedID := args[1]

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		if err := svc.AddDependency(blockerID, blockedID); err != nil {
			output.Error(err)
		}

		output.Success("dependency added", map[string]any{
			"blocker": blockerID,
			"blocked": blockedID,
		})
		return nil
	},
}

var depRemoveCmd = &cobra.Command{
	Use:   "remove <blocker-id> <blocked-id>",
	Short: "Remove a dependency",
	Long:  `Removes a blocking relationship between two tasks.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		blockerID := args[0]
		blockedID := args[1]

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		if err := svc.RemoveDependency(blockerID, blockedID); err != nil {
			output.Error(err)
		}

		output.Success("dependency removed", map[string]any{
			"blocker": blockerID,
			"blocked": blockedID,
		})
		return nil
	},
}

var depListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List dependencies for a task",
	Long:  `Shows what tasks block the given task and what tasks it blocks.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		t, err := svc.GetTaskByID(taskID)
		if err != nil {
			output.Error(err)
		}

		output.JSON(map[string]any{
			"task_id":    taskID,
			"blocked_by": t.BlockedBy(),
			"blocks":     t.Blocks(),
		})
		return nil
	},
}

// Flags for dep tree command
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

		// Validate direction flag
		if treeDirection != "down" && treeDirection != "up" && treeDirection != "both" {
			output.Error(fmt.Errorf("invalid direction: %s (valid: down, up, both)", treeDirection))
		}

		// Validate status flag if provided
		var filterStatus *task.Status
		if treeStatus != "" {
			s, err := task.ParseStatus(treeStatus)
			if err != nil {
				output.Error(err)
			}
			filterStatus = &s
		}

		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		// Load all tasks to build the full dependency graph
		tasks, err := svc.LoadAllTasks()
		if err != nil {
			output.Error(err)
		}

		// Build task map for quick lookup
		taskMap := make(map[string]task.Task)
		for _, t := range tasks {
			taskMap[t.ID()] = t
		}

		// Verify the target task exists
		rootTask, exists := taskMap[taskID]
		if !exists {
			output.Error(fmt.Errorf("task not found: %s", taskID))
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

var depChainCmd = &cobra.Command{
	Use:   "chain <id1> <id2> [id3] ...",
	Short: "Create a chain of dependencies",
	Long: `Creates sequential dependencies between tasks.
The first task blocks the second, the second blocks the third, and so on.

Example:
  pace task dep chain pace-001 pace-002 pace-003
  Creates: pace-001 blocks pace-002, pace-002 blocks pace-003`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := task.NewService()
		if err != nil {
			output.Error(err)
		}
		defer svc.Close()

		var dependencies []map[string]string
		var errors []string

		// Create sequential dependencies
		for i := 0; i < len(args)-1; i++ {
			blockerID := args[i]
			blockedID := args[i+1]

			if err := svc.AddDependency(blockerID, blockedID); err != nil {
				errors = append(errors, fmt.Sprintf("%s->%s: %s", blockerID, blockedID, err.Error()))
			} else {
				dependencies = append(dependencies, map[string]string{
					"blocker": blockerID,
					"blocked": blockedID,
				})
			}
		}

		if len(errors) > 0 && len(dependencies) == 0 {
			output.ErrorMsg(strings.Join(errors, "; "))
		}

		data := map[string]any{
			"dependencies": dependencies,
		}
		if len(errors) > 0 {
			data["errors"] = errors
		}

		output.Success("dependency chain created", data)
		return nil
	},
}

func init() {
	depCmd.AddCommand(depAddCmd)
	depCmd.AddCommand(depRemoveCmd)
	depCmd.AddCommand(depListCmd)
	depCmd.AddCommand(depTreeCmd)
	depCmd.AddCommand(depChainCmd)

	// Tree command flags
	depTreeCmd.Flags().StringVar(&treeDirection, "direction", "up", "Tree direction: 'up' (blockers), 'down' (blocks), or 'both'")
	depTreeCmd.Flags().StringVar(&treeStatus, "status", "", "Filter by status (todo, in-progress, done)")
	depTreeCmd.Flags().IntVarP(&treeMaxDepth, "max-depth", "d", 50, "Maximum tree depth to display")
}

// Styles for the dependency tree
var (
	treeRootStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	treeBlockerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	treeBlocksStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	treeNodeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	treeBranchStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	treeLabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	treeReadyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
)

// treeOptions holds configuration for tree printing
type treeOptions struct {
	direction    string
	filterStatus *task.Status
	maxDepth     int
	taskMap      map[string]task.Task
}

// printDepTree prints an ASCII tree visualization of task dependencies
func printDepTree(root task.Task, opts treeOptions) {
	fmt.Println()

	showBlockers := opts.direction == "up" || opts.direction == "both"
	showBlocks := opts.direction == "down" || opts.direction == "both"

	// Print blockers section (what blocks this task)
	blockers := root.BlockedBy()
	if showBlockers && len(blockers) > 0 {
		fmt.Println(treeBlockerStyle.Render("BLOCKED BY:"))
		printTree(blockers, opts, "", make(map[string]bool), true, 0)
		fmt.Println()
	}

	// Print the root task
	fmt.Print(treeRootStyle.Render(fmt.Sprintf("► %s: %s", root.ID(), root.Title())))
	// Show [READY] indicator if task has no unresolved blockers
	if isTaskReady(root, opts.taskMap) {
		fmt.Print(" " + treeReadyStyle.Render("[READY]"))
	}
	fmt.Println()
	printTaskStatus(root)
	fmt.Println()

	// Print blocks section (what this task blocks)
	blocks := root.Blocks()
	if showBlocks && len(blocks) > 0 {
		fmt.Println(treeBlocksStyle.Render("BLOCKS:"))
		printTree(blocks, opts, "", make(map[string]bool), false, 0)
		fmt.Println()
	}

	// If no dependencies in the requested direction, print a message
	hasBlockers := showBlockers && len(blockers) > 0
	hasBlocks := showBlocks && len(blocks) > 0
	if !hasBlockers && !hasBlocks {
		fmt.Println(treeLabelStyle.Render("No dependencies found."))
		fmt.Println()
	}
}

// isTaskReady checks if a task has no unresolved blockers
func isTaskReady(t task.Task, taskMap map[string]task.Task) bool {
	if t.Status() == task.Done {
		return false // Done tasks aren't "ready"
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

// printTaskStatus prints the status of a task in a compact format
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
		}
		fmt.Printf(" %s", pStyle.Render(fmt.Sprintf("P%d", p)))
	}
	fmt.Println()
}

// printTree recursively prints tasks in a tree structure
func printTree(taskIDs []string, opts treeOptions, prefix string, visited map[string]bool, isBlocker bool, depth int) {
	// Filter task IDs based on status if filter is set
	var filteredIDs []string
	for _, id := range taskIDs {
		if t, exists := opts.taskMap[id]; exists {
			if opts.filterStatus == nil || t.Status() == *opts.filterStatus {
				filteredIDs = append(filteredIDs, id)
			}
		} else {
			// Keep non-existent tasks to show "not found" message
			filteredIDs = append(filteredIDs, id)
		}
	}

	for i, id := range filteredIDs {
		isLast := i == len(filteredIDs)-1
		printTreeNode(id, opts, prefix, isLast, visited, isBlocker, depth)
	}
}

// printTreeNode prints a single node in the tree and recursively prints children
func printTreeNode(id string, opts treeOptions, prefix string, isLast bool, visited map[string]bool, isBlocker bool, depth int) {
	// Check max depth before printing (depth is 0-indexed, maxDepth=1 means show 1 level)
	if depth >= opts.maxDepth {
		return
	}

	// Determine branch characters
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
		// Task doesn't exist (might have been deleted)
		fmt.Printf("%s%s%s\n",
			treeBranchStyle.Render(prefix+branch),
			treeLabelStyle.Render(id),
			treeLabelStyle.Render(" (not found)"))
		return
	}

	// Check for cycles
	if visited[id] {
		fmt.Printf("%s%s%s\n",
			treeBranchStyle.Render(prefix+branch),
			treeNodeStyle.Render(fmt.Sprintf("%s: %s", id, t.Title())),
			treeLabelStyle.Render(" (cycle)"))
		return
	}

	// Format status indicator
	var statusIndicator string
	switch t.Status() {
	case task.Todo:
		statusIndicator = todoStyle.Render("○")
	case task.InProgress:
		statusIndicator = progressStyle.Render("●")
	case task.Done:
		statusIndicator = doneStyle.Render("●")
	}

	// Build the node line
	nodeLine := fmt.Sprintf("%s: %s", id, truncateTitle(t.Title(), 50))

	// Add [READY] indicator for ready tasks
	if isTaskReady(t, opts.taskMap) {
		nodeLine += " " + treeReadyStyle.Render("[READY]")
	}

	// Print the node
	fmt.Printf("%s%s %s\n",
		treeBranchStyle.Render(prefix+branch),
		statusIndicator,
		treeNodeStyle.Render(nodeLine))

	// Mark as visited to detect cycles
	visited[id] = true

	// Recursively print children
	var children []string
	if isBlocker {
		children = t.BlockedBy() // Go up the blocker chain
	} else {
		children = t.Blocks() // Go down the blocks chain
	}

	if len(children) > 0 {
		printTree(children, opts, childPrefix, visited, isBlocker, depth+1)
	}

	// Unmark for other branches (allow same task to appear in different branches)
	delete(visited, id)
}

// truncateTitle truncates a title if it exceeds maxLen
func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return strings.TrimSpace(title[:maxLen-3]) + "..."
}
