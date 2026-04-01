package mcp

import (
	"encoding/json"
	"testing"
)

func TestJSONRPCRequest_MarshalRoundTrip(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      42,
		Method:  MethodToolsCall,
		Params: ToolCallParams{
			Name:      "Bash",
			Arguments: map[string]interface{}{"command": "echo hi"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got JSONRPCRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q", got.JSONRPC)
	}
	if got.ID != 42 {
		t.Errorf("id = %d", got.ID)
	}
	if got.Method != MethodToolsCall {
		t.Errorf("method = %q", got.Method)
	}
}

func TestJSONRPCResponse_WithResult(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]interface{}{"ok": true},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got JSONRPCResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Error != nil {
		t.Error("error should be nil for success response")
	}
	if got.Result == nil {
		t.Error("result should not be nil")
	}
}

func TestJSONRPCResponse_WithError(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      2,
		Error:   &JSONRPCError{Code: -32600, Message: "invalid request"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got JSONRPCResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Error == nil {
		t.Fatal("error should not be nil")
	}
	if got.Error.Code != -32600 {
		t.Errorf("code = %d", got.Error.Code)
	}
	if got.Error.Message != "invalid request" {
		t.Errorf("message = %q", got.Error.Message)
	}
}

func TestJSONRPCError_ErrorInterface(t *testing.T) {
	e := &JSONRPCError{Code: -32601, Message: "method not found"}
	if e.Error() != "method not found" {
		t.Errorf("Error() = %q", e.Error())
	}
}

func TestMethodConstants(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{MethodInitialize, "initialize"},
		{MethodToolsList, "tools/list"},
		{MethodToolsCall, "tools/call"},
		{MethodResourceList, "resources/list"},
		{MethodResourceRead, "resources/read"},
		{MethodPromptsList, "prompts/list"},
		{MethodPromptsGet, "prompts/get"},
		{MethodShutdown, "shutdown"},
	}
	for _, tt := range tests {
		if tt.name != tt.want {
			t.Errorf("constant = %q, want %q", tt.name, tt.want)
		}
	}
}

func TestInitializeParams_Marshal(t *testing.T) {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      ClientInfo{Name: "ti-code", Version: "0.1.0"},
		Capabilities:    ClientCapabilities{Tools: &ToolCapabilities{}},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got InitializeParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ProtocolVersion != "2024-11-05" {
		t.Errorf("version = %q", got.ProtocolVersion)
	}
	if got.ClientInfo.Name != "ti-code" {
		t.Errorf("client name = %q", got.ClientInfo.Name)
	}
}

func TestToolsListResult_Marshal(t *testing.T) {
	result := ToolsListResult{
		Tools: []MCPToolSchema{
			{Name: "bash", Description: "run commands"},
			{Name: "read", Description: "read files"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ToolsListResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.Tools) != 2 {
		t.Fatalf("got %d tools, want 2", len(got.Tools))
	}
	if got.Tools[0].Name != "bash" {
		t.Errorf("first tool = %q", got.Tools[0].Name)
	}
}

func TestToolCallResult_Marshal(t *testing.T) {
	result := ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: "hello"}},
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ToolCallResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.Content) != 1 || got.Content[0].Text != "hello" {
		t.Errorf("content mismatch: %+v", got.Content)
	}
	if got.IsError {
		t.Error("IsError should be false")
	}
}

func TestResourceListResult_Marshal(t *testing.T) {
	result := ResourceListResult{
		Resources: []MCPResource{
			{URI: "file:///test.txt", Name: "test", MimeType: "text/plain"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ResourceListResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.Resources) != 1 {
		t.Fatalf("got %d resources", len(got.Resources))
	}
	if got.Resources[0].URI != "file:///test.txt" {
		t.Errorf("uri = %q", got.Resources[0].URI)
	}
}
