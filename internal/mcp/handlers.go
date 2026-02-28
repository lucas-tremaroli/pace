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
	case "pace_info":
		return h.toolInfo()
	case "pace_task_list":
		return h.toolTaskList(args)
	case "pace_task_create":
		return h.toolTaskCreate(args)
	case "pace_task_update":
		return h.toolTaskUpdate(args)
	case "pace_task_delete":
		return h.toolTaskDelete(args)
	case "pace_task_ready":
		return h.toolTaskReady()
	case "pace_task_dep_add":
		return h.toolTaskDepAdd(args)
	case "pace_task_dep_remove":
		return h.toolTaskDepRemove(args)
	case "pace_note_list":
		return h.toolNoteList(args)
	case "pace_note_create":
		return h.toolNoteCreate(args)
	case "pace_note_read":
		return h.toolNoteRead(args)
	case "pace_note_delete":
		return h.toolNoteDelete(args)
	default:
		return ToolCallResult{
			Content: []ContentBlock{NewTextContent(fmt.Sprintf("unknown tool: %s", name))},
			IsError: true,
		}
	}
}

func (h *Handler) toolInfo() ToolCallResult {
	resolved, err := storage.ResolvePaceDir()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to resolve storage: %v", err))
	}

	tasks, err := h.taskService.LoadAllTasks()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to load tasks: %v", err))
	}

	notes, err := h.noteService.ListNotes()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list notes: %v", err))
	}

	// Count tasks by status
	var todoCount, inProgressCount, doneCount int
	for _, t := range tasks {
		switch t.Status() {
		case task.Todo:
			todoCount++
		case task.InProgress:
			inProgressCount++
		case task.Done:
			doneCount++
		}
	}

	info := map[string]any{
		"storage": map[string]any{
			"path": resolved.Path,
			"type": string(resolved.Type),
		},
		"tasks": map[string]any{
			"total":       len(tasks),
			"todo":        todoCount,
			"in_progress": inProgressCount,
			"done":        doneCount,
		},
		"notes": map[string]any{
			"total": len(notes),
		},
	}

	return jsonResult(info)
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
			v, ok := p.(float64)
			if !ok {
				return codedError(output.ErrCodeInvalidPriority, "priority values must be numbers", "Valid values: 1 (urgent), 2 (high), 3 (normal), 4 (low)")
			}
			priority := int(v)
			if float64(priority) != v {
				return codedError(output.ErrCodeInvalidPriority, "priority values must be integers", "Valid values: 1 (urgent), 2 (high), 3 (normal), 4 (low)")
			}
			if priority < 1 || priority > 4 {
				return codedError(output.ErrCodeInvalidPriority, "priority values must be between 1 and 4", "Valid values: 1 (urgent), 2 (high), 3 (normal), 4 (low)")
			}
			filter.Priorities = append(filter.Priorities, priority)
		}
	}
	if labels, ok := args["label"].([]any); ok {
		for _, l := range labels {
			if s, ok := l.(string); ok && s != "" {
				filter.AnyLabels = append(filter.AnyLabels, s)
			}
		}
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

	// Parse type
	taskType := task.TypeTask
	if typeStr, ok := args["type"].(string); ok && typeStr != "" {
		var err error
		taskType, err = task.ParseTaskType(typeStr)
		if err != nil {
			return codedError(output.ErrCodeInvalidType, err.Error(), "Valid values: task, bug, feature, chore, docs")
		}
	}

	// Parse priority
	priority := 3
	if p, ok := args["priority"].(float64); ok {
		if p != float64(int(p)) {
			return codedError(output.ErrCodeInvalidPriority, "priority must be an integer", "Valid values: 1 (urgent), 2 (high), 3 (normal), 4 (low)")
		}
		pi := int(p)
		if pi < 1 || pi > 4 {
			return codedError(output.ErrCodeInvalidPriority, "priority must be between 1 and 4", "Valid values: 1 (urgent), 2 (high), 3 (normal), 4 (low)")
		}
		priority = pi
	}

	// Generate ID and create task
	id := h.taskService.GenerateTaskID()
	newTask := task.NewTaskComplete(id, status, taskType, title, description, priority, link)

	if err := h.taskService.CreateTask(newTask); err != nil {
		return errorResult(fmt.Sprintf("failed to create task: %v", err))
	}

	// Add labels if provided
	if labelsRaw, ok := args["labels"]; ok {
		if labels, ok := labelsRaw.([]any); ok {
			for _, l := range labels {
				if label, ok := l.(string); ok && label != "" {
					if err := h.taskService.AddLabel(id, label); err != nil {
						return errorResult(fmt.Sprintf("task %s created but failed to add label %q: %v", id, label, err))
					}
				}
			}
		}
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

	taskType := existingTask.Type()
	if t, ok := args["type"].(string); ok && t != "" {
		taskType, err = task.ParseTaskType(t)
		if err != nil {
			return codedError(output.ErrCodeInvalidType, err.Error(), "Valid values: task, bug, feature, chore, docs")
		}
	}

	priority := existingTask.Priority()
	if p, ok := args["priority"].(float64); ok {
		if p != float64(int(p)) {
			return codedError(output.ErrCodeInvalidPriority, "priority must be an integer", "Valid values: 1 (urgent), 2 (high), 3 (normal), 4 (low)")
		}
		pi := int(p)
		if pi < 1 || pi > 4 {
			return codedError(output.ErrCodeInvalidPriority, "priority must be between 1 and 4", "Valid values: 1 (urgent), 2 (high), 3 (normal), 4 (low)")
		}
		priority = pi
	}

	updatedTask := task.NewTaskComplete(id, status, taskType, title, description, priority, link)
	if err := h.taskService.UpdateTask(updatedTask); err != nil {
		return errorResult(fmt.Sprintf("failed to update task: %v", err))
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

func (h *Handler) toolTaskReady() ToolCallResult {
	tasks, err := h.taskService.GetReadyTasks()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get ready tasks: %v", err))
	}

	taskList := make([]task.TaskJSON, 0, len(tasks))
	for _, t := range tasks {
		taskList = append(taskList, t.ToJSON())
	}

	return jsonResult(map[string]any{"tasks": taskList})
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
			"mod_time":    n.ModTime,
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

	return jsonResult(map[string]any{
		"filename":    noteData.Filename,
		"content":     noteData.Content,
		"description": noteData.Description,
		"labels":      noteData.Labels,
		"mod_time":    noteData.ModTime,
	})
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
