package mcp

import (
	"encoding/json"
	"testing"
)

func TestNewTextContent(t *testing.T) {
	content := NewTextContent("hello world")
	if content.Type != "text" {
		t.Errorf("expected type 'text', got '%s'", content.Type)
	}
	if content.Text != "hello world" {
		t.Errorf("expected text 'hello world', got '%s'", content.Text)
	}
}

func TestNewErrorResponse(t *testing.T) {
	id := json.RawMessage(`1`)
	resp := NewErrorResponse(id, MethodNotFound, "method not found")

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got '%s'", resp.JSONRPC)
	}
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != MethodNotFound {
		t.Errorf("expected code %d, got %d", MethodNotFound, resp.Error.Code)
	}
	if resp.Error.Message != "method not found" {
		t.Errorf("expected message 'method not found', got '%s'", resp.Error.Message)
	}
}

func TestNewSuccessResponse(t *testing.T) {
	id := json.RawMessage(`1`)
	result := map[string]string{"key": "value"}
	resp := NewSuccessResponse(id, result)

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got '%s'", resp.JSONRPC)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestGetToolDefinitions(t *testing.T) {
	tools := GetToolDefinitions()
	if len(tools) == 0 {
		t.Fatal("expected tools, got empty list")
	}

	// Check that required tools exist
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{
		"pace_context",
		"pace_task_list",
		"pace_task_create",
		"pace_task_update",
		"pace_task_delete",
		"pace_task_dep_add",
		"pace_task_dep_remove",
		"pace_note_list",
		"pace_note_create",
		"pace_note_read",
		"pace_note_delete",
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("missing expected tool: %s", name)
		}
	}
}
