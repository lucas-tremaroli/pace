package mcp

// GetToolDefinitions returns all available MCP tools
func GetToolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "pace_info",
			Description: "Get project overview including storage location, task counts by status, and notes count",
			InputSchema: ToolInputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "pace_task_list",
			Description: "List all tasks with their status, type, priority, dependencies, and labels",
			InputSchema: ToolInputSchema{
				Type:       "object",
				Properties: map[string]Property{},
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
					"type": {
						Type:        "string",
						Description: "Task type",
						Enum:        []string{"task", "bug", "feature", "chore", "docs"},
						Default:     "task",
					},
					"priority": {
						Type:        "integer",
						Description: "Priority level: 1 (urgent), 2 (high), 3 (normal), 4 (low)",
						Default:     3,
					},
					"labels": {
						Type:        "array",
						Description: "List of label strings to attach to the task",
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
					"type": {
						Type:        "string",
						Description: "New task type",
						Enum:        []string{"task", "bug", "feature", "chore", "docs"},
					},
					"priority": {
						Type:        "integer",
						Description: "New priority level: 1 (urgent), 2 (high), 3 (normal), 4 (low)",
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
			Name:        "pace_task_ready",
			Description: "Get tasks that are ready to work on (not blocked by other tasks)",
			InputSchema: ToolInputSchema{
				Type:       "object",
				Properties: map[string]Property{},
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
			Name:        "pace_note_list",
			Description: "List all notes with their filenames, descriptions, and labels",
			InputSchema: ToolInputSchema{
				Type:       "object",
				Properties: map[string]Property{},
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
	}
}
