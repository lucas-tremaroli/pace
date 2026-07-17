package mcp

import "github.com/lucas-tremaroli/pace/internal/task"

// GetToolDefinitions returns all available MCP tools
func GetToolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "pace_context",
			Description: "Get storage location, active tasks (todo/in-progress), and notes for session context",
			InputSchema: ToolInputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "pace_task_list",
			Description: "List all tasks with their status, priority, dependencies, and label. Supports optional filtering by status, priority, and label.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"status": {
						Type:        "string",
						Description: "Filter by status",
						Enum:        []string{"todo", "in-progress", "done"},
					},
					"priority": {
						Type:        "array",
						Description: "Filter by one or more priority levels: 1 (high), 2 (medium), 3 (low). Accepts names (high, medium, low) or numbers (1-3)",
						Items: &Property{
							Type: "string",
						},
					},
					"label": {
						Type:        "string",
						Description: "Filter by label",
					},
					"fields": {
						Type:        "array",
						Description: "Limit response to specific fields. Available: id, title, description, status, priority, blocked_by, blocks, label, notes, link",
						Items: &Property{
							Type: "string",
						},
					},
					"head": {
						Type:        "integer",
						Description: "Limit output to first N tasks",
					},
					"ready": {
						Type:        "boolean",
						Description: "Only show tasks ready to work on (not blocked, not done)",
					},
				},
			},
		},
		{
			Name:        "pace_task_get",
			Description: "Get a single task by ID with its dependencies and label",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Task ID (required)",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "pace_task_create",
			Description: "Create a new task",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title": {
						Type:        "string",
						Description: "Task title (required)",
					},
					"description": {
						Type:        "string",
						Description: "Task description",
					},
					"status": {
						Type:        "string",
						Description: "Task status",
						Enum:        []string{"todo", "in-progress", "done"},
						Default:     "todo",
					},
					"priority": {
						Type:        "string",
						Description: "Priority level: high (1), medium (2), low (3). Accepts names or numbers",
						Default:     "low",
					},
					"label": {
						Type:        "string",
						Description: "Task label",
						Enum:        task.ValidLabels,
						Default:     task.LabelTask,
					},
					"link": {
						Type:        "string",
						Description: "URL link associated with the task",
					},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "pace_task_update",
			Description: "Update an existing task",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Task ID (required)",
					},
					"title": {
						Type:        "string",
						Description: "New task title",
					},
					"description": {
						Type:        "string",
						Description: "New task description",
					},
					"status": {
						Type:        "string",
						Description: "New task status",
						Enum:        []string{"todo", "in-progress", "done"},
					},
					"priority": {
						Type:        "string",
						Description: "New priority level: high (1), medium (2), low (3). Accepts names or numbers",
					},
					"label": {
						Type:        "string",
						Description: "New task label",
						Enum:        task.ValidLabels,
					},
					"link": {
						Type:        "string",
						Description: "New URL link (use empty string to clear)",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "pace_task_delete",
			Description: "Delete a task by ID",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Task ID to delete (required)",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "pace_task_dep_add",
			Description: "Create a blocking dependency between tasks (blocker blocks blocked)",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"blocker_id": {
						Type:        "string",
						Description: "ID of the task that blocks another",
					},
					"blocked_id": {
						Type:        "string",
						Description: "ID of the task that is blocked",
					},
				},
				Required: []string{"blocker_id", "blocked_id"},
			},
		},
		{
			Name:        "pace_task_dep_remove",
			Description: "Remove a blocking dependency between tasks",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"blocker_id": {
						Type:        "string",
						Description: "ID of the task that blocks another",
					},
					"blocked_id": {
						Type:        "string",
						Description: "ID of the task that is blocked",
					},
				},
				Required: []string{"blocker_id", "blocked_id"},
			},
		},
		{
			Name:        "pace_task_note_link",
			Description: "Link a note to a task",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"task_id": {
						Type:        "string",
						Description: "Task ID to link the note to",
					},
					"note_filename": {
						Type:        "string",
						Description: "Note filename (with or without .md extension)",
					},
				},
				Required: []string{"task_id", "note_filename"},
			},
		},
		{
			Name:        "pace_task_note_unlink",
			Description: "Unlink a note from a task",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"task_id": {
						Type:        "string",
						Description: "Task ID to unlink the note from",
					},
					"note_filename": {
						Type:        "string",
						Description: "Note filename (with or without .md extension)",
					},
				},
				Required: []string{"task_id", "note_filename"},
			},
		},
		{
			Name:        "pace_note_list",
			Description: "List all notes with their filenames, descriptions, and labels",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"fields": {
						Type:        "array",
						Description: "Limit response to specific fields. Available: filename, description, labels, modTime",
						Items: &Property{
							Type: "string",
						},
					},
					"head": {
						Type:        "integer",
						Description: "Limit output to first N notes",
					},
				},
			},
		},
		{
			Name:        "pace_note_create",
			Description: "Create a new note",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"filename": {
						Type:        "string",
						Description: "Note filename without .md extension (required)",
					},
					"content": {
						Type:        "string",
						Description: "Note content in markdown format (required)",
					},
					"task_ids": {
						Type:        "array",
						Description: "Task IDs to link to this note",
						Items: &Property{
							Type: "string",
						},
					},
				},
				Required: []string{"filename", "content"},
			},
		},
		{
			Name:        "pace_note_read",
			Description: "Read a note's content and metadata",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"filename": {
						Type:        "string",
						Description: "Note filename (with or without .md extension)",
					},
				},
				Required: []string{"filename"},
			},
		},
		{
			Name:        "pace_note_delete",
			Description: "Delete a note",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"filename": {
						Type:        "string",
						Description: "Note filename to delete",
					},
				},
				Required: []string{"filename"},
			},
		},
		{
			Name:        "pace_task_log",
			Description: "Add a progress log entry to a task",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Task ID (required)",
					},
					"message": {
						Type:        "string",
						Description: "Log message to record (required)",
					},
				},
				Required: []string{"id", "message"},
			},
		},
		{
			Name:        "pace_task_close",
			Description: "Close a task (mark as done) with an optional outcome message",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Task ID (required)",
					},
					"outcome": {
						Type:        "string",
						Description: "Outcome message to record",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "pace_task_logs",
			Description: "View log entries for a task",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Task ID (required)",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "pace_task_bulk_delete",
			Description: "Delete multiple tasks at once by IDs or status filter",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"ids": {
						Type:        "array",
						Description: "List of task IDs to delete",
						Items: &Property{
							Type: "string",
						},
					},
					"status": {
						Type:        "string",
						Description: "Delete all tasks with this status",
						Enum:        []string{"todo", "in-progress", "done"},
					},
				},
			},
		},
		{
			Name:        "pace_epic_create",
			Description: "Create a new epic — a spec-first container that groups tasks",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title": {
						Type:        "string",
						Description: "Epic title (required)",
					},
					"summary": {
						Type:        "string",
						Description: "Short summary of the epic",
					},
					"status": {
						Type:        "string",
						Description: "Epic status (default: planning)",
						Enum:        []string{"planning", "active", "done"},
					},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "pace_epic_list",
			Description: "List all epics with their status and spec",
			InputSchema: ToolInputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "pace_epic_get",
			Description: "Get a single epic by ID, including its full spec",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Epic ID (required)",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "pace_epic_update",
			Description: "Update an epic's title, summary, or status. Only provided fields change.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Epic ID (required)",
					},
					"title": {
						Type:        "string",
						Description: "New title",
					},
					"summary": {
						Type:        "string",
						Description: "New summary",
					},
					"status": {
						Type:        "string",
						Description: "New status",
						Enum:        []string{"planning", "active", "done"},
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "pace_epic_delete",
			Description: "Delete an epic. Tasks that belonged to it are kept but unlinked.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Epic ID (required)",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "pace_epic_spec_set",
			Description: "Set one or more spec sections on an epic (current/target/constraints/exclusions plus a freeform fallback). Only provided sections change.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Epic ID (required)",
					},
					"current_state": {
						Type:        "string",
						Description: "Current state section (markdown)",
					},
					"target_state": {
						Type:        "string",
						Description: "Target state section (markdown)",
					},
					"constraints": {
						Type:        "string",
						Description: "Constraints section (markdown)",
					},
					"exclusions": {
						Type:        "string",
						Description: "Exclusions section (markdown)",
					},
					"freeform": {
						Type:        "string",
						Description: "Freeform fallback section (markdown)",
					},
				},
				Required: []string{"id"},
			},
		},
	}
}
