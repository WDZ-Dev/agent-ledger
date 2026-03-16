package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Request is a JSON-RPC 2.0 request or notification.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ToolCallParams holds the parameters from a tools/call request.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// InitializeResult holds the result of an initialize response.
type InitializeResult struct {
	ServerInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

// ParseMessage parses a JSON-RPC 2.0 message and returns whether it is a
// request or response. Notifications (requests without an id) are returned
// as requests.
func ParseMessage(data []byte) (isRequest bool, req *Request, resp *Response, err error) {
	// Quick check for valid JSON.
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return false, nil, nil, fmt.Errorf("empty message")
	}

	// Peek at the object to determine type. A request has "method", a
	// response has "result" or "error".
	var probe struct {
		Method string          `json:"method"`
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return false, nil, nil, fmt.Errorf("invalid JSON-RPC: %w", err)
	}

	if probe.Method != "" {
		var r Request
		if err := json.Unmarshal(data, &r); err != nil {
			return false, nil, nil, fmt.Errorf("parsing request: %w", err)
		}
		return true, &r, nil, nil
	}

	if probe.Result != nil || probe.Error != nil {
		var r Response
		if err := json.Unmarshal(data, &r); err != nil {
			return false, nil, nil, fmt.Errorf("parsing response: %w", err)
		}
		return false, nil, &r, nil
	}

	return false, nil, nil, fmt.Errorf("message is neither request nor response")
}
