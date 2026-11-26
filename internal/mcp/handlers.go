package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/output"
	"github.com/lucas-tremaroli/pace/internal/storage"
	"github.com/lucas-tremaroli/pace/internal/task"
)

// Version is set by the main package
var Version = "dev"

// Handler manages MCP request handling
type Handler struct {
	taskService *task.Service
	epicService *epic.Service
}

// NewHandler creates a new MCP handler with initialized services
func NewHandler() (*Handler, error) {
	taskService, err := task.NewService()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task service: %w", err)
	}

	epicService, err := epic.NewService()
	if err != nil {
		taskService.Close()
		return nil, fmt.Errorf("failed to initialize epic service: %w", err)
	}

	return &Handler{
		taskService: taskService,
		epicService: epicService,
	}, nil
}

// Close cleans up handler resources, closing every service and returning a
// combined error so no close failure is silently dropped.
func (h *Handler) Close() error {
	var errs []error
	if h.epicService != nil {
		errs = append(errs, h.epicService.Close())
	}
	if h.taskService != nil {
		errs = append(errs, h.taskService.Close())
	}
	return errors.Join(errs...)
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

	activeEpics, err := h.epicService.LoadOpenEpics()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to load epics: %v", err))
	}
	epicList := make([]map[string]any, 0, len(activeEpics))
	for _, e := range activeEpics {
		epicList = append(epicList, activeEpicSummary(e))
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
		"active_epics": epicList,
		"tasks": map[string]any{
			"in_progress": inProgressList,
			"todo":        todoList,
		},
		"summary": map[string]any{
			"tasks": map[string]any{
				"total":       len(tasks),
				"todo":        len(todoList),
				"in_progress": len(inProgressList),
				"done":        len(tasks) - len(todoList) - len(inProgressList),
			},
			"epics": len(epicList),
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
				if float64(priority) != v || !task.IsValidPriority(priority) {
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

	return jsonResult(output.TaskListResponse{Tasks: taskList, Count: len(taskList)})
}

func (h *Handler) toolTaskGet(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the task ID to retrieve")
	}

	t, err := h.taskService.GetTaskByID(id)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
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
	label := task.LabelTask
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
			if float64(priority) != v || !task.IsValidPriority(priority) {
				return codedError(output.ErrCodeInvalidPriority, "priority must be 1-3", task.ValidPriorityHelp)
			}
		default:
			return codedError(output.ErrCodeInvalidPriority, "priority must be a string or number", task.ValidPriorityHelp)
		}
	}

	// Generate ID and create task
	id := h.taskService.GenerateTaskID()
	newTask := task.NewTaskComplete(id, status, title, description, priority, link)
	if epicID, ok := args["epic_id"].(string); ok {
		newTask.SetEpicID(epicID)
	}

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
		if errors.Is(err, task.ErrTaskNotFound) {
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
			if float64(priority) != v || !task.IsValidPriority(priority) {
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

	// Assign (or clear, with "") the epic if provided
	if epicID, ok := args["epic_id"].(string); ok {
		if err := h.taskService.SetEpic(id, epicID); err != nil {
			return errorResult(fmt.Sprintf("task updated but failed to set epic: %v", err))
		}
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
		if errors.Is(err, task.ErrTaskNotFound) {
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
		if errors.Is(err, task.ErrTaskNotFound) {
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
		if errors.Is(err, task.ErrTaskNotFound) {
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
		if errors.Is(err, task.ErrTaskNotFound) {
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
			if errors.Is(err, task.ErrTaskNotFound) {
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

// --- Epic tools ---

func (h *Handler) toolEpicCreate(args map[string]any) ToolCallResult {
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return codedError(output.ErrCodeMissingField, "title is required", "Provide a title string when creating an epic")
	}
	summary, _ := args["summary"].(string)

	status := epic.Planning
	if s, ok := args["status"].(string); ok && s != "" {
		parsed, err := epic.ParseStatus(s)
		if err != nil {
			return codedError(output.ErrCodeInvalidStatus, err.Error(), "Valid values: planning, active, done")
		}
		status = parsed
	}

	e := epic.NewEpic(h.epicService.GenerateEpicID(), status, title, summary)
	e.SetSpec(epic.Spec{
		CurrentState: stringArg(args, "current_state"),
		TargetState:  stringArg(args, "target_state"),
		Constraints:  stringArg(args, "constraints"),
		Exclusions:   stringArg(args, "exclusions"),
		Freeform:     stringArg(args, "freeform"),
	})
	if labels, ok := stringSliceArg(args, "labels"); ok {
		e.SetLabels(labels)
	}
	if err := h.epicService.CreateEpic(e); err != nil {
		return errorResult(fmt.Sprintf("failed to create epic: %v", err))
	}
	return jsonResult(map[string]any{
		"success": true,
		"message": fmt.Sprintf("created epic %s", e.ID()),
		"epic":    e.ToJSON(),
	})
}

func (h *Handler) toolEpicList(args map[string]any) ToolCallResult {
	epics, err := h.epicService.LoadAllEpics()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list epics: %v", err))
	}
	if labels, ok := stringSliceArg(args, "labels"); ok && len(labels) > 0 {
		var filtered []epic.Epic
		for _, e := range epics {
			for _, l := range labels {
				if e.HasLabel(l) {
					filtered = append(filtered, e)
					break
				}
			}
		}
		epics = filtered
	}
	if head := parseHead(args); head > 0 && head < len(epics) {
		epics = epics[:head]
	}
	jsons := make([]epic.EpicJSON, 0, len(epics))
	for _, e := range epics {
		jsons = append(jsons, e.ToJSON())
	}
	return jsonResult(map[string]any{"success": true, "epics": jsons, "count": len(jsons)})
}

func (h *Handler) toolEpicGet(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the epic ID to retrieve")
	}
	e, err := h.epicService.GetEpicByID(id)
	if err != nil {
		return epicMCPLookupError(err, id)
	}
	return jsonResult(map[string]any{"success": true, "epic": e.ToJSON()})
}

func (h *Handler) toolEpicUpdate(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the epic ID to update")
	}
	e, err := h.epicService.GetEpicByID(id)
	if err != nil {
		return epicMCPLookupError(err, id)
	}

	if title, ok := args["title"].(string); ok && title != "" {
		e.SetTitle(title)
	}
	if summary, ok := args["summary"].(string); ok {
		e.SetSummary(summary)
	}
	if labels, ok := stringSliceArg(args, "labels"); ok {
		e.SetLabels(labels)
	}
	if s, ok := args["status"].(string); ok && s != "" {
		parsed, err := epic.ParseStatus(s)
		if err != nil {
			return codedError(output.ErrCodeInvalidStatus, err.Error(), "Valid values: planning, active, done")
		}
		if err := e.SetStatus(parsed); err != nil {
			return errorResult(err.Error())
		}
	}

	if err := h.epicService.UpdateEpic(*e); err != nil {
		return errorResult(fmt.Sprintf("failed to update epic: %v", err))
	}
	return jsonResult(map[string]any{"success": true, "message": fmt.Sprintf("updated epic %s", id), "epic": e.ToJSON()})
}

func (h *Handler) toolEpicDelete(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the epic ID to delete")
	}
	if err := h.epicService.DeleteEpic(id); err != nil {
		return epicMCPLookupError(err, id)
	}
	return jsonResult(map[string]any{"success": true, "message": fmt.Sprintf("deleted epic %s", id)})
}

func (h *Handler) toolEpicSpecSet(args map[string]any) ToolCallResult {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return codedError(output.ErrCodeMissingField, "id is required", "Provide the epic ID whose spec to set")
	}
	e, err := h.epicService.GetEpicByID(id)
	if err != nil {
		return epicMCPLookupError(err, id)
	}

	changed := false
	if v, ok := args["current_state"].(string); ok {
		e.SetCurrentState(v)
		changed = true
	}
	if v, ok := args["target_state"].(string); ok {
		e.SetTargetState(v)
		changed = true
	}
	if v, ok := args["constraints"].(string); ok {
		e.SetConstraints(v)
		changed = true
	}
	if v, ok := args["exclusions"].(string); ok {
		e.SetExclusions(v)
		changed = true
	}
	if v, ok := args["freeform"].(string); ok {
		e.SetFreeform(v)
		changed = true
	}
	if !changed {
		return codedError(output.ErrCodeInvalidParams, "no spec sections provided", "Pass at least one of current_state, target_state, constraints, exclusions, freeform")
	}

	if err := h.epicService.UpdateEpic(*e); err != nil {
		return errorResult(fmt.Sprintf("failed to update epic spec: %v", err))
	}
	return jsonResult(map[string]any{"success": true, "message": fmt.Sprintf("updated spec for epic %s", id), "epic": e.ToJSON()})
}

func epicMCPLookupError(err error, id string) ToolCallResult {
	if errors.Is(err, epic.ErrEpicNotFound) {
		return codedError(output.ErrCodeEpicNotFound, fmt.Sprintf("epic not found: %s", id), "Use pace_epic_list to see available epic IDs")
	}
	return codedError(output.ErrCodeStorageError, fmt.Sprintf("epic operation failed: %v", err), "")
}

// activeEpicSummary returns an id/title/status plus spec headlines for context.
func activeEpicSummary(e epic.Epic) map[string]any {
	spec := e.Spec()
	return map[string]any{
		"id":            e.ID(),
		"title":         e.Title(),
		"status":        e.Status().String(),
		"summary":       e.Summary(),
		"current_state": headline(spec.CurrentState),
		"target_state":  headline(spec.TargetState),
	}
}

// headline trims a spec section to its first non-empty line for compact display.
func headline(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
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

func stringArg(args map[string]any, key string) string {
	s, _ := args[key].(string)
	return s
}

// stringSliceArg reads a JSON string array argument. ok is false when the key
// is absent, so callers can tell "not provided" from "clear".
func stringSliceArg(args map[string]any, key string) ([]string, bool) {
	raw, ok := args[key].([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out, true
}
