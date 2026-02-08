package output

import (
	"encoding/json"
	"os"
)

// Response represents a standard JSON response
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// JSON prints any value as formatted JSON to stdout
func JSON(v any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(v)
}

// Success prints a success response with optional data
func Success(message string, data any) {
	JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error prints an error response and exits with code 1
func Error(err error) {
	JSON(Response{
		Success: false,
		Error:   err.Error(),
	})
	os.Exit(1)
}

// ErrorMsg prints an error message response and exits with code 1
func ErrorMsg(message string) {
	JSON(Response{
		Success: false,
		Error:   message,
	})
	os.Exit(1)
}

// BulkResult represents the result of a bulk operation
type BulkResult struct {
	Succeeded []BulkItem `json:"succeeded"`
	Failed    []BulkItem `json:"failed"`
	Total     int        `json:"total"`
}

// BulkItem represents a single item in a bulk operation result
type BulkItem struct {
	ID       string   `json:"id,omitempty"`
	Title    string   `json:"title,omitempty"`
	Error    string   `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// BulkSuccess prints a bulk operation result
// Returns success if at least one operation succeeded
func BulkSuccess(message string, result BulkResult) {
	success := len(result.Succeeded) > 0
	resp := Response{
		Success: success,
		Message: message,
		Data:    result,
	}
	if !success && len(result.Failed) > 0 {
		resp.Error = "all operations failed"
	}
	JSON(resp)
	if !success {
		os.Exit(1)
	}
}
