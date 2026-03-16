package mcp

import (
	"encoding/json"
	"testing"
)

func TestParseMessage_Request(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_file"}}`)

	isReq, req, resp, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isReq {
		t.Fatal("expected request")
	}
	if req == nil {
		t.Fatal("expected non-nil request")
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
	if req.Method != "tools/call" {
		t.Errorf("method = %q, want %q", req.Method, "tools/call")
	}
}

func TestParseMessage_Response(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"hello"}]}}`)

	isReq, req, resp, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isReq {
		t.Fatal("expected response, not request")
	}
	if req != nil {
		t.Fatal("expected nil request")
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Error != nil {
		t.Error("expected nil error")
	}
}

func TestParseMessage_ErrorResponse(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":2,"error":{"code":-32601,"message":"method not found"}}`)

	isReq, _, resp, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isReq {
		t.Fatal("expected response")
	}
	if resp.Error == nil {
		t.Fatal("expected non-nil error")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want %d", resp.Error.Code, -32601)
	}
}

func TestParseMessage_Notification(t *testing.T) {
	// Notifications have no id field.
	data := []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)

	isReq, req, _, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isReq {
		t.Fatal("expected request (notification)")
	}
	if req.Method != "notifications/initialized" {
		t.Errorf("method = %q, want %q", req.Method, "notifications/initialized")
	}
	if req.ID != nil {
		t.Error("expected nil ID for notification")
	}
}

func TestParseMessage_StringID(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":"abc-123","method":"tools/list"}`)

	_, req, _, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(req.ID) != `"abc-123"` {
		t.Errorf("ID = %s, want %q", req.ID, `"abc-123"`)
	}
}

func TestParseMessage_EmptyInput(t *testing.T) {
	_, _, _, err := ParseMessage([]byte(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseMessage_MalformedJSON(t *testing.T) {
	_, _, _, err := ParseMessage([]byte("{not json"))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseMessage_NeitherRequestNorResponse(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1}`)
	_, _, _, err := ParseMessage(data)
	if err == nil {
		t.Fatal("expected error for ambiguous message")
	}
}

func TestToolCallParams_Parse(t *testing.T) {
	params := json.RawMessage(`{"name":"read_file","arguments":{"path":"/tmp"}}`)
	var tcp ToolCallParams
	if err := json.Unmarshal(params, &tcp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tcp.Name != "read_file" {
		t.Errorf("name = %q, want %q", tcp.Name, "read_file")
	}
}

func TestInitializeResult_Parse(t *testing.T) {
	data := []byte(`{"serverInfo":{"name":"filesystem","version":"1.0.0"}}`)
	var ir InitializeResult
	if err := json.Unmarshal(data, &ir); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ir.ServerInfo.Name != "filesystem" {
		t.Errorf("server name = %q, want %q", ir.ServerInfo.Name, "filesystem")
	}
	if ir.ServerInfo.Version != "1.0.0" {
		t.Errorf("server version = %q, want %q", ir.ServerInfo.Version, "1.0.0")
	}
}
