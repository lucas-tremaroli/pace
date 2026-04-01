package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Server implements the MCP server using JSON-RPC over stdio
type Server struct {
	handler *Handler
	reader  *bufio.Reader
	writer  io.Writer
}

// NewServer creates a new MCP server
func NewServer() (*Server, error) {
	handler, err := NewHandler()
	if err != nil {
		return nil, err
	}

	return &Server{
		handler: handler,
		reader:  bufio.NewReader(os.Stdin),
		writer:  os.Stdout,
	}, nil
}

// Close cleans up server resources
func (s *Server) Close() error {
	if s.handler != nil {
		return s.handler.Close()
	}
	return nil
}

// Run starts the server loop, reading from stdin and writing to stdout
func (s *Server) Run() error {
	defer s.Close()

	for {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading stdin: %w", err)
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := NewErrorResponse(nil, ParseError, "parse error")
			s.writeResponse(resp)
			continue
		}

		// Validate JSON-RPC version
		if req.JSONRPC != "2.0" {
			resp := NewErrorResponse(req.ID, InvalidRequest, "invalid jsonrpc version")
			s.writeResponse(resp)
			continue
		}

		// Handle the request
		resp := s.handler.HandleRequest(req)
		if resp != nil {
			s.writeResponse(*resp)
		}
	}
}

func (s *Server) writeResponse(resp JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal response: %v\n", err)
		return
	}
	fmt.Fprintf(s.writer, "%s\n", data)
}
