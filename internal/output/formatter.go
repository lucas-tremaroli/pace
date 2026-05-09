package output

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

// FilterFields filters a slice of maps to only include the specified fields.
// If fields is nil or empty, all fields are returned unchanged.
func FilterFields(items []map[string]any, fields []string) []map[string]any {
	if len(fields) == 0 {
		return items
	}
	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[strings.TrimSpace(f)] = true
	}
	result := make([]map[string]any, len(items))
	for i, item := range items {
		filtered := make(map[string]any, len(fields))
		for k, v := range item {
			if fieldSet[k] {
				filtered[k] = v
			}
		}
		result[i] = filtered
	}
	return result
}

// ToMapSlice marshals a slice of structs to []map[string]any using JSON field names.
func ToMapSlice(v any) ([]map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Error code constants for machine-readable error identification
const (
	ErrCodeTaskNotFound       = "TASK_NOT_FOUND"
	ErrCodeNoteNotFound       = "NOTE_NOT_FOUND"
	ErrCodeMissingField       = "MISSING_REQUIRED_FIELD"
	ErrCodeInvalidStatus      = "INVALID_STATUS"
	ErrCodeInvalidType        = "INVALID_TYPE"
	ErrCodeInvalidPriority    = "INVALID_PRIORITY"
	ErrCodeInvalidFilename    = "INVALID_FILENAME"
	ErrCodeInvalidParams      = "INVALID_PARAMS"
	ErrCodeStorageError       = "STORAGE_ERROR"
)

// Response represents a standard JSON response
type Response struct {
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
	ErrorCode  string `json:"error_code,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
	Data       any    `json:"data,omitempty"`
}

// JSON prints any value as compact JSON to stdout
func JSON(v any) {
	encoder := json.NewEncoder(os.Stdout)
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

// Error prints a JSON error response and returns err.
func Error(err error) error {
	JSON(Response{
		Success: false,
		Error:   err.Error(),
	})
	return err
}

// ErrorMsg prints a JSON error response and returns an error with the given message.
func ErrorMsg(message string) error {
	JSON(Response{
		Success: false,
		Error:   message,
	})
	return errors.New(message)
}

// ErrorWithCode prints a JSON error response with a machine-readable code and returns err.
func ErrorWithCode(err error, code, suggestion string) error {
	JSON(Response{
		Success:    false,
		Error:      err.Error(),
		ErrorCode:  code,
		Suggestion: suggestion,
	})
	return err
}

// ErrorMsgWithCode prints a JSON error response with a machine-readable code and returns an error.
func ErrorMsgWithCode(message, code, suggestion string) error {
	JSON(Response{
		Success:    false,
		Error:      message,
		ErrorCode:  code,
		Suggestion: suggestion,
	})
	return errors.New(message)
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
