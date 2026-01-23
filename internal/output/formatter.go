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
