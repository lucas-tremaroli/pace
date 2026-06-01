package mcp

import "fmt"

// toolHandler is a method on Handler that processes a tool call's arguments.
type toolHandler func(*Handler, map[string]any) ToolCallResult

// toolRegistry maps MCP tool names to their handler methods.
// Adding a new tool is a single registry entry — no switch edit required.
var toolRegistry = map[string]toolHandler{
	"pace_context":          func(h *Handler, _ map[string]any) ToolCallResult { return h.toolContext() },
	"pace_task_list":        (*Handler).toolTaskList,
	"pace_task_get":         (*Handler).toolTaskGet,
	"pace_task_create":      (*Handler).toolTaskCreate,
	"pace_task_update":      (*Handler).toolTaskUpdate,
	"pace_task_delete":      (*Handler).toolTaskDelete,
	"pace_task_dep_add":     (*Handler).toolTaskDepAdd,
	"pace_task_dep_remove":  (*Handler).toolTaskDepRemove,
	"pace_task_note_link":   (*Handler).toolTaskNoteLink,
	"pace_task_note_unlink": (*Handler).toolTaskNoteUnlink,
	"pace_note_list":        (*Handler).toolNoteList,
	"pace_note_create":      (*Handler).toolNoteCreate,
	"pace_note_read":        (*Handler).toolNoteRead,
	"pace_note_delete":      (*Handler).toolNoteDelete,
	"pace_task_log":         (*Handler).toolTaskLog,
	"pace_task_close":       (*Handler).toolTaskClose,
	"pace_task_logs":        (*Handler).toolTaskLogs,
	"pace_task_bulk_delete": (*Handler).toolTaskBulkDelete,
}

func (h *Handler) executeTool(name string, args map[string]any) ToolCallResult {
	handler, ok := toolRegistry[name]
	if !ok {
		return ToolCallResult{
			Content: []ContentBlock{NewTextContent(fmt.Sprintf("unknown tool: %s", name))},
			IsError: true,
		}
	}
	return handler(h, args)
}
