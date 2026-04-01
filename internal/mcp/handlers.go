package mcp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/note"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/lucas-tremaroli/pace/internal/task"
)

// Version is set by the main package
var Version = "dev"

// Handler manages MCP request handling
type Handler struct {
	taskService *task.Service
	noteService *note.Service
}

// NewHandler creates a new MCP handler with initialized services
func NewHandler() (*Handler, error) {
	taskService, err := task.NewService()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task service: %w", err)
	}

	noteService, err := note.NewService()
	if err != nil {
		taskService.Close()
		return nil, fmt.Errorf("failed to initialize note service: %w", err)
	}

	return &Handler{
		taskService: taskService,
		noteService: noteService,
	}, nil
}

// Close cleans up handler resources
func (h *Handler) Close() error {
	if h.taskService != nil {
		return h.taskService.Close()
	}
	return nil
}

// HandleRequest processes a JSON-RPC request and returns a response
func (h *Handler) HandleRequest(req JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return h.handleInitialize(req)
	case "notifications/initialized":
		// This is a notification, no response needed
		return nil
	case "tools/list":
		return h.handleToolsList(req)
	case "tools/call":
		return h.handleToolsCall(req)
	default:
		resp := NewErrorResponse(req.ID, MethodNotFound, fmt.Sprintf("method not found: %s", req.Method))
		return &resp
	}
}

func (h *Handler) handleInitialize(req JSONRPCRequest) *JSONRPCResponse {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "pace",
			Version: Version,
		},
	}
	resp := NewSuccessResponse(req.ID, result)
	return &resp
}

func (h *Handler) handleToolsList(req JSONRPCRequest) *JSONRPCResponse {
	result := ToolsListResult{
		Tools: GetToolDefinitions(),
	}
	resp := NewSuccessResponse(req.ID, result)
	return &resp
}

func (h *Handler) handleToolsCall(req JSONRPCRequest) *JSONRPCResponse {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		resp := NewErrorResponse(req.ID, InvalidParams, "invalid params")
		return &resp
	}

	result := h.executeTool(params.Name, params.Arguments)
	resp := NewSuccessResponse(req.ID, result)
	return &resp
}

func (h *Handler) executeTool(name string, args map[string]any) ToolCallResult {
	switch name {
	case "pace_context":
		return h.toolContext()
	case "pace_task_list":
		return h.toolTaskList(args)
	case "pace_task_get":
		return h.toolTaskGet(args)
	case "pace_task_create":
		return h.toolTaskCreate(args)
	case "pace_task_update":
		return h.toolTaskUpdate(args)
	case "pace_task_delete":
		return h.toolTaskDelete(args)
	case "pace_task_dep_add":
		return h.toolTaskDepAdd(args)
	case "pace_task_dep_remove":
		return h.toolTaskDepRemove(args)
	case "pace_task_note_link":
		return h.toolTaskNoteLink(args)
	case "pace_task_note_unlink":
		return h.toolTaskNoteUnlink(args)
	case "pace_note_list":
		return h.toolNoteList(args)
	case "pace_note_create":
		return h.toolNoteCreate(args)
	case "pace_note_read":
		return h.toolNoteRead(args)
	case "pace_note_delete":
		return h.toolNoteDelete(args)
	case "pace_task_log":
		return h.toolTaskLog(args)
	case "pace_task_close":
		return h.toolTaskClose(args)
	case "pace_task_logs":
		return h.toolTaskLogs(args)
	case "pace_task_bulk_delete":
		return h.toolTaskBulkDelete(args)
	default:
		return ToolCallResult{
			Content: []ContentBlock{NewTextContent(fmt.Sprintf("unknown tool: %s", name))},
			IsError: true,
		}
	}
}

func (h *Handler) toolContext() ToolCallResult {
	tasks, err := h.taskService.LoadAllTasks()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to load tasks: %v", err))
	}

	todoStatus := task.Todo
	inProgressStatus := task.InProgress
	todoFilter := task.TaskFilter{Status: &todoStatus}
	inProgressFilter := task.TaskFilter{Status: &inProgressStatus}

	todoTasks := todoFilter.Apply(tasks)
	inProgressTasks := inProgressFilter.Apply(tasks)

	todoList := make([]task.TaskJSON, 0, len(todoTasks))
	for _, t := range todoTasks {
		todoList = append(todoList, t.ToJSON())
	}

	inProgressList := make([]task.TaskJSON, 0, len(inProgressTasks))
	for _, t := range inProgressTasks {
		inProgressList = append(inProgressList, t.ToJSON())
	}

	notes, err := h.noteService.ListNotesWithMeta(false)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list notes: %v", err))
	}

	noteList := make([]map[string]any, 0, len(notes))
	for _, n := range notes {
		noteList = append(noteList, map[string]any{
			"filename":    n.Filename,
			"description": n.Description,
			"labels":      n.Labels,
		})
	}

	resolved, err := storage.ResolvePaceDir()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to resolve storage: %v", err))
	}

	return jsonResult(map[string]any{
		"storage": map[string]any{
			"path": resolved.Path,
			"type": string(resolved.Type),
		},
		"tasks": map[string]any{
			"in_progress": inProgressList,
			"todo":        todoList,
		},
		"notes": noteList,
		"summary": map[string]any{
			"tasks": map[string]any{
				"total":       len(tasks),
				"todo":        len(todoList),
				"in_progress": len(inProgressList),
				"done":        len(tasks) - len(todoList) - len(inProgressList),
			},
			"notes": len(noteList),
		},
	})
}

func (h *Handler) toolTaskList(args map[string]any) ToolCallResult {
	tasks, err := h.taskService.LoadAllTasks()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to load tasks: %v", err))
	}

	// Build and apply filter
	filter := task.TaskFilter{}
	if statusStr, ok := args["status"].(string); ok && statusStr != "" {
		status, err := task.ParseStatus(statusStr)
		if err != nil {
			return codedError(output.ErrCodeInvalidStatus, err.Error(), "Valid values: todo, in-progress, done")
		}
		filter.Status = &status
	}
	if priorities, ok := args["priority"].([]any); ok {
		for _, p := range priorities {
			var priority int
			switch v := p.(type) {
			case string:
				var err error
				priority, err = task.ParsePriority(v)
				if err != nil {
					return codedError(output.ErrCodeInvalidPriority, err.Error(), task.ValidPriorityHelp)
				}
			case float64:
				priority = int(v)
				if float64(priority) != v || priority < 1 || priority > 3 {
					return codedError(output.ErrCodeInvalidPriority, "priority must be 1-3", task.ValidPriorityHelp)
				}
			default:
				return codedError(output.ErrCodeInvalidPriority, "priority values must be strings or numbers", task.ValidPriorityHelp)
			}
			filter.Priorities = append(filter.Priorities, priority)
		}
	}
	if label, ok := args["label"].(string); ok && label != "" {
		filter.Label = &label
	}
	if ready, ok := args["ready"].(bool); ok && ready {
		filter.Ready = true
	}
	tasks = filter.Apply(tasks)

	// Apply head truncation
	if head := parseHead(args); head > 0 && head < len(tasks) {
		tasks = tasks[:head]
	}

	taskList := make([]task.TaskJSON, 0, len(tasks))
	for _, t := range tasks {
		taskList = append(taskList, t.ToJSON())
	}

	// Apply field filtering
	if fields := parseFields(args); len(fields) > 0 {
		maps, err := output.ToMapSlice(taskList)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to filter fields: %v", err))
		}
		filtered := output.FilterFields(maps, fields)
		return jsonResult(map[string]any{"tasks": filtered, "count": len(filtered)})
	}

	return jsonResult(map[string]any{"tasks": taskList, "count": len(taskList)})
}

func (h *Handler) toolTaskGet(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the task ID to retrieve")
	}

	t, err := h.taskService.GetTaskByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return codedError(output.ErrCodeTaskNotFound, fmt.Sprintf("task not found: %s", id), "Use pace_task_list to see available task IDs")
		}
		return codedError(output.ErrCodeStorageError, fmt.Sprintf("failed to get task: %v", err), "")
	}

	return jsonResult(map[string]any{
		"success": true,
		"task":    t.ToJSON(),
	})
}

func (h *Handler) toolTaskCreate(args map[string]any) ToolCallResult {
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return codedError(output.ErrCodeMissingField, "title is required", "Provide a title string when creating a task")
	}

	description, _ := args["description"].(string)
	link, _ := args["link"].(string)

	// Parse status
	status := task.Todo
	if statusStr, ok := args["status"].(string); ok && statusStr != "" {
		var err error
		status, err = task.ParseStatus(statusStr)
		if err != nil {
			return codedError(output.ErrCodeInvalidStatus, err.Error(), "Valid values: todo, in-progress, done")
		}
	}

	// Parse label
	label := "task"
	if l, ok := args["label"].(string); ok && l != "" {
		if err := task.ValidateLabel(l); err != nil {
			return codedError(output.ErrCodeInvalidType, err.Error(), "Valid values: task, bug, feature, docs")
		}
		label = l
	}

	// Parse priority
	priority := 3
	if p, ok := args["priority"]; ok {
		switch v := p.(type) {
		case string:
			var err error
			priority, err = task.ParsePriority(v)
			if err != nil {
				return codedError(output.ErrCodeInvalidPriority, err.Error(), task.ValidPriorityHelp)
			}
		case float64:
			priority = int(v)
			if float64(priority) != v || priority < 1 || priority > 3 {
				return codedError(output.ErrCodeInvalidPriority, "priority must be 1-3", task.ValidPriorityHelp)
			}
		default:
			return codedError(output.ErrCodeInvalidPriority, "priority must be a string or number", task.ValidPriorityHelp)
		}
	}

	// Generate ID and create task
	id := h.taskService.GenerateTaskID()
	newTask := task.NewTaskComplete(id, status, title, description, priority, link)

	if err := h.taskService.CreateTask(newTask); err != nil {
		return errorResult(fmt.Sprintf("failed to create task: %v", err))
	}

	// Set label
	if err := h.taskService.SetLabel(id, label); err != nil {
		return errorResult(fmt.Sprintf("task %s created but failed to set label %q: %v", id, label, err))
	}

	// Reload task to get full data
	createdTask, err := h.taskService.GetTaskByID(id)
	if err != nil {
		return errorResult(fmt.Sprintf("task created but failed to reload: %v", err))
	}

	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("created task %s", id),
		"task":    createdTask.ToJSON(),
	})
}

func (h *Handler) toolTaskUpdate(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the task ID to update")
	}

	// Load existing task
	existingTask, err := h.taskService.GetTaskByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return codedError(output.ErrCodeTaskNotFound, fmt.Sprintf("task not found: %s", id), "Use pace_task_list to see available task IDs")
		}
		return codedError(output.ErrCodeStorageError, fmt.Sprintf("failed to load task %s: %v", id, err), "")
	}

	// Build updated task with existing values as defaults
	title := existingTask.Title()
	if t, ok := args["title"].(string); ok {
		title = t
	}

	description := existingTask.Description()
	if d, ok := args["description"].(string); ok {
		description = d
	}

	link := existingTask.Link()
	if l, ok := args["link"].(string); ok {
		link = l
	}

	status := existingTask.Status()
	if s, ok := args["status"].(string); ok && s != "" {
		status, err = task.ParseStatus(s)
		if err != nil {
			return codedError(output.ErrCodeInvalidStatus, err.Error(), "Valid values: todo, in-progress, done")
		}
	}

	priority := existingTask.Priority()
	if p, ok := args["priority"]; ok {
		switch v := p.(type) {
		case string:
			var err error
			priority, err = task.ParsePriority(v)
			if err != nil {
				return codedError(output.ErrCodeInvalidPriority, err.Error(), task.ValidPriorityHelp)
			}
		case float64:
			priority = int(v)
			if float64(priority) != v || priority < 1 || priority > 3 {
				return codedError(output.ErrCodeInvalidPriority, "priority must be 1-3", task.ValidPriorityHelp)
			}
		default:
			return codedError(output.ErrCodeInvalidPriority, "priority must be a string or number", task.ValidPriorityHelp)
		}
	}

	updatedTask := task.NewTaskComplete(id, status, title, description, priority, link)
	if err := h.taskService.UpdateTask(updatedTask); err != nil {
		return errorResult(fmt.Sprintf("failed to update task: %v", err))
	}

	// Set label if provided
	if l, ok := args["label"].(string); ok {
		if err := task.ValidateLabel(l); err != nil {
			return codedError(output.ErrCodeInvalidType, err.Error(), "Valid values: task, bug, feature, docs")
		}
		if err := h.taskService.SetLabel(id, l); err != nil {
			return errorResult(fmt.Sprintf("task updated but failed to set label: %v", err))
		}
	}

	// Reload task to get full data
	reloadedTask, err := h.taskService.GetTaskByID(id)
	if err != nil {
		return errorResult(fmt.Sprintf("task updated but failed to reload: %v", err))
	}

	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("updated task %s", id),
		"task":    reloadedTask.ToJSON(),
	})
}

func (h *Handler) toolTaskDelete(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the task ID to delete")
	}

	if err := h.taskService.DeleteTask(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return codedError(output.ErrCodeTaskNotFound, fmt.Sprintf("task not found: %s", id), "Use pace_task_list to see available task IDs")
		}
		return codedError(output.ErrCodeStorageError, fmt.Sprintf("failed to delete task: %v", err), "")
	}

	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("deleted task %s", id),
	})
}

func (h *Handler) toolTaskDepAdd(args map[string]any) ToolCallResult {
	blockerID, ok := args["blocker_id"].(string)
	if !ok || blockerID == "" {
		return codedError(output.ErrCodeMissingField, "blocker_id is required", "Provide the ID of the task that should block another")
	}

	blockedID, ok := args["blocked_id"].(string)
	if !ok || blockedID == "" {
		return codedError(output.ErrCodeMissingField, "blocked_id is required", "Provide the ID of the task that should be blocked")
	}

	if err := h.taskService.AddDependency(blockerID, blockedID); err != nil {
		return errorResult(fmt.Sprintf("failed to add dependency: %v", err))
	}

	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("%s now blocks %s", blockerID, blockedID),
	})
}

func (h *Handler) toolTaskDepRemove(args map[string]any) ToolCallResult {
	blockerID, ok := args["blocker_id"].(string)
	if !ok || blockerID == "" {
		return codedError(output.ErrCodeMissingField, "blocker_id is required", "Provide the ID of the blocking task")
	}

	blockedID, ok := args["blocked_id"].(string)
	if !ok || blockedID == "" {
		return codedError(output.ErrCodeMissingField, "blocked_id is required", "Provide the ID of the blocked task")
	}

	if err := h.taskService.RemoveDependency(blockerID, blockedID); err != nil {
		return errorResult(fmt.Sprintf("failed to remove dependency: %v", err))
	}

	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("%s no longer blocks %s", blockerID, blockedID),
	})
}

func (h *Handler) toolTaskNoteLink(args map[string]any) ToolCallResult {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return codedError(output.ErrCodeMissingField, "task_id is required", "Provide the task ID to link the note to")
	}

	noteFilename, ok := args["note_filename"].(string)
	if !ok || noteFilename == "" {
		return codedError(output.ErrCodeMissingField, "note_filename is required", "Provide the note filename")
	}

	// Normalize filename to include .md extension
	if !strings.HasSuffix(noteFilename, ".md") {
		noteFilename += ".md"
	}

	if err := h.taskService.LinkNote(taskID, noteFilename); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return codedError(output.ErrCodeTaskNotFound, fmt.Sprintf("task not found: %s", taskID), "Use pace_task_list to see available task IDs")
		}
		return errorResult(fmt.Sprintf("failed to link note: %v", err))
	}

	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("linked %s to task %s", noteFilename, taskID),
	})
}

func (h *Handler) toolTaskNoteUnlink(args map[string]any) ToolCallResult {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return codedError(output.ErrCodeMissingField, "task_id is required", "Provide the task ID to unlink the note from")
	}

	noteFilename, ok := args["note_filename"].(string)
	if !ok || noteFilename == "" {
		return codedError(output.ErrCodeMissingField, "note_filename is required", "Provide the note filename")
	}

	// Normalize filename to include .md extension
	if !strings.HasSuffix(noteFilename, ".md") {
		noteFilename += ".md"
	}

	if err := h.taskService.UnlinkNote(taskID, noteFilename); err != nil {
		return errorResult(fmt.Sprintf("failed to unlink note: %v", err))
	}

	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("unlinked %s from task %s", noteFilename, taskID),
	})
}

func (h *Handler) toolNoteList(args map[string]any) ToolCallResult {
	notes, err := h.noteService.ListNotesWithMeta(false)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list notes: %v", err))
	}

	// Apply head truncation
	if head := parseHead(args); head > 0 && head < len(notes) {
		notes = notes[:head]
	}

	noteList := make([]map[string]any, 0, len(notes))
	for _, n := range notes {
		noteList = append(noteList, map[string]any{
			"filename":    n.Filename,
			"description": n.Description,
			"labels":      n.Labels,
			"modTime":    n.ModTime,
		})
	}

	// Apply field filtering
	if fields := parseFields(args); len(fields) > 0 {
		filtered := output.FilterFields(noteList, fields)
		return jsonResult(map[string]any{"notes": filtered, "count": len(filtered)})
	}

	return jsonResult(map[string]any{"notes": noteList, "count": len(noteList)})
}

func (h *Handler) toolNoteCreate(args map[string]any) ToolCallResult {
	filename, ok := args["filename"].(string)
	if !ok || filename == "" {
		return codedError(output.ErrCodeMissingField, "filename is required", "Provide a filename without the .md extension")
	}
	if err := validateFilename(filename); err != nil {
		return codedError(output.ErrCodeInvalidFilename, err.Error(), "Use a simple filename without path separators (e.g. my-note)")
	}

	content, ok := args["content"].(string)
	if !ok {
		return codedError(output.ErrCodeMissingField, "content is required", "Provide the note content as a markdown string")
	}

	if err := h.noteService.WriteNote(filename, content); err != nil {
		return codedError(output.ErrCodeStorageError, fmt.Sprintf("failed to create note: %v", err), "")
	}

	// Ensure filename has .md extension for display
	displayName := filename
	if !strings.HasSuffix(displayName, ".md") {
		displayName += ".md"
	}

	// Link to tasks if task_ids provided
	if taskIDsRaw, ok := args["task_ids"]; ok {
		if taskIDs, ok := taskIDsRaw.([]any); ok {
			for _, tid := range taskIDs {
				if id, ok := tid.(string); ok && id != "" {
					if err := h.taskService.LinkNote(id, displayName); err != nil {
						return errorResult(fmt.Sprintf("note %s created but failed to link to task %s: %v", displayName, id, err))
					}
				}
			}
		}
	}

	return jsonResult(map[string]any{
		"success":  true,
		"message":  fmt.Sprintf("created note %s", displayName),
		"filename": displayName,
	})
}

func (h *Handler) toolNoteRead(args map[string]any) ToolCallResult {
	filename, ok := args["filename"].(string)
	if !ok || filename == "" {
		return codedError(output.ErrCodeMissingField, "filename is required", "Provide the note filename (with or without .md extension)")
	}
	if err := validateFilename(filename); err != nil {
		return codedError(output.ErrCodeInvalidFilename, err.Error(), "Use a simple filename without path separators (e.g. my-note)")
	}

	noteData, err := h.noteService.ReadNoteWithMeta(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return codedError(output.ErrCodeNoteNotFound, fmt.Sprintf("note not found: %s", filename), "Use pace_note_list to see available notes")
		}
		return codedError(output.ErrCodeStorageError, fmt.Sprintf("failed to read note: %v", err), "")
	}

	result := map[string]any{
		"filename":    noteData.Filename,
		"content":     noteData.Content,
		"description": noteData.Description,
		"labels":      noteData.Labels,
		"modTime":     noteData.ModTime,
	}
	if len(noteData.Tasks) > 0 {
		result["tasks"] = noteData.Tasks
	}
	return jsonResult(result)
}

func (h *Handler) toolNoteDelete(args map[string]any) ToolCallResult {
	filename, ok := args["filename"].(string)
	if !ok || filename == "" {
		return codedError(output.ErrCodeMissingField, "filename is required", "Provide the note filename to delete")
	}
	if err := validateFilename(filename); err != nil {
		return codedError(output.ErrCodeInvalidFilename, err.Error(), "Use a simple filename without path separators (e.g. my-note)")
	}

	// Ensure filename has .md extension
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	if err := h.noteService.DeleteNote(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return codedError(output.ErrCodeNoteNotFound, fmt.Sprintf("note not found: %s", filename), "Use pace_note_list to see available notes")
		}
		return codedError(output.ErrCodeStorageError, fmt.Sprintf("failed to delete note: %v", err), "")
	}

	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("deleted note %s", filename),
	})
}

func (h *Handler) toolTaskLog(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the task ID")
	}

	message, ok := args["message"].(string)
	if !ok || message == "" {
		return codedError(output.ErrCodeMissingField, "message is required", "Provide a log message")
	}

	if err := h.taskService.LogEntry(id, message); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return codedError(output.ErrCodeTaskNotFound, fmt.Sprintf("task not found: %s", id), "Use pace_task_list to see available task IDs")
		}
		return errorResult(fmt.Sprintf("failed to add log: %v", err))
	}

	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("log entry added to %s", id),
	})
}

func (h *Handler) toolTaskClose(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the task ID to close")
	}

	outcome, _ := args["outcome"].(string)

	if err := h.taskService.CloseTask(id, outcome); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return codedError(output.ErrCodeTaskNotFound, fmt.Sprintf("task not found: %s", id), "Use pace_task_list to see available task IDs")
		}
		return errorResult(fmt.Sprintf("failed to close task: %v", err))
	}

	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("closed task %s", id),
	})
}

func (h *Handler) toolTaskLogs(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the task ID")
	}

	logs, err := h.taskService.GetTaskLogs(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return codedError(output.ErrCodeTaskNotFound, fmt.Sprintf("task not found: %s", id), "Use pace_task_list to see available task IDs")
		}
		return errorResult(fmt.Sprintf("failed to get logs: %v", err))
	}

	return jsonResult(map[string]any{
		"task_id": id,
		"logs":    logs,
		"count":   len(logs),
	})
}

func (h *Handler) toolTaskBulkDelete(args map[string]any) ToolCallResult {
	var ids []string

	// Collect IDs from explicit list
	if idsRaw, ok := args["ids"].([]any); ok {
		for _, id := range idsRaw {
			if s, ok := id.(string); ok && s != "" {
				ids = append(ids, s)
			}
		}
	}

	// Collect IDs from status filter
	if statusStr, ok := args["status"].(string); ok && statusStr != "" {
		status, err := task.ParseStatus(statusStr)
		if err != nil {
			return codedError(output.ErrCodeInvalidStatus, err.Error(), "Valid values: todo, in-progress, done")
		}

		tasks, err := h.taskService.LoadAllTasks()
		if err != nil {
			return errorResult(fmt.Sprintf("failed to load tasks: %v", err))
		}

		filter := task.TaskFilter{Status: &status}
		for _, t := range filter.Apply(tasks) {
			ids = append(ids, t.ID())
		}
	}

	if len(ids) == 0 {
		return codedError(output.ErrCodeMissingField, "no tasks to delete", "Provide ids array and/or status filter")
	}

	// Deduplicate
	seen := make(map[string]bool, len(ids))
	unique := ids[:0]
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	var deleted []string
	var errs []string
	for _, id := range unique {
		if err := h.taskService.DeleteTask(id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errs = append(errs, fmt.Sprintf("%s: not found", id))
			} else {
				errs = append(errs, fmt.Sprintf("%s: %v", id, err))
			}
			continue
		}
		deleted = append(deleted, id)
	}

	result := map[string]any{
		"success":       len(errs) == 0,
		"deleted":       deleted,
		"deleted_count": len(deleted),
	}
	if len(errs) > 0 {
		result["errors"] = errs
	}
	return jsonResult(result)
}

// Helper functions

func jsonResult(data any) ToolCallResult {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal result: %v", err))
	}
	return ToolCallResult{
		Content: []ContentBlock{NewTextContent(string(jsonBytes))},
	}
}

func errorResult(message string) ToolCallResult {
	return ToolCallResult{
		Content: []ContentBlock{NewTextContent(message)},
		IsError: true,
	}
}

func codedError(code, message, suggestion string) ToolCallResult {
	data, _ := json.Marshal(map[string]any{
		"error":      message,
		"error_code": code,
		"suggestion": suggestion,
	})
	return ToolCallResult{
		Content: []ContentBlock{NewTextContent(string(data))},
		IsError: true,
	}
}

// parseFields extracts the "fields" array from MCP args.
func parseFields(args map[string]any) []string {
	fieldsRaw, ok := args["fields"].([]any)
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(fieldsRaw))
	for _, f := range fieldsRaw {
		if s, ok := f.(string); ok && s != "" {
			fields = append(fields, strings.TrimSpace(s))
		}
	}
	return fields
}

// parseHead extracts the "head" integer from MCP args.
func parseHead(args map[string]any) int {
	if head, ok := args["head"].(float64); ok && head > 0 {
		if float64(int(head)) == head {
			return int(head)
		}
	}
	return 0
}

// validateFilename rejects filenames that could escape the notes directory.
func validateFilename(filename string) error {
	if filepath.IsAbs(filename) {
		return fmt.Errorf("filename must not be an absolute path")
	}
	if filepath.Base(filename) != filename {
		return fmt.Errorf("filename must not contain path separators")
	}
	return nil
}
