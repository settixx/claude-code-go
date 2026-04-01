package mcp

// Standard JSON-RPC method names used by the MCP protocol.
const (
	MethodInitialize   = "initialize"
	MethodToolsList    = "tools/list"
	MethodToolsCall    = "tools/call"
	MethodResourceList = "resources/list"
	MethodResourceRead = "resources/read"
	MethodShutdown     = "shutdown"
)

// JSONRPCRequest is a JSON-RPC 2.0 request envelope.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response envelope.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError carries structured error information inside a JSON-RPC response.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string { return e.Message }

// InitializeParams are sent with the "initialize" request.
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}

// ClientInfo identifies the MCP client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities advertises what the client supports.
type ClientCapabilities struct {
	Tools     *ToolCapabilities     `json:"tools,omitempty"`
	Resources *ResourceCapabilities `json:"resources,omitempty"`
}

// ToolCapabilities describes client-side tool support.
type ToolCapabilities struct{}

// ResourceCapabilities describes client-side resource support.
type ResourceCapabilities struct{}

// InitializeResult is the server's response to "initialize".
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
}

// ServerInfo identifies the MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities advertises what the server supports.
type ServerCapabilities struct {
	Tools     interface{} `json:"tools,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
	Prompts   interface{} `json:"prompts,omitempty"`
}

// MCPToolSchema describes a tool as returned by "tools/list".
type MCPToolSchema struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
}

// ToolsListResult is the server's response to "tools/list".
type ToolsListResult struct {
	Tools []MCPToolSchema `json:"tools"`
}

// ToolCallParams are sent with the "tools/call" request.
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResult is the server's response to "tools/call".
type ToolCallResult struct {
	Content []ToolContent `json:"content,omitempty"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent is a single content block inside a tool-call result.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// MCPResource describes a resource as returned by "resources/list".
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceListResult is the server's response to "resources/list".
type ResourceListResult struct {
	Resources []MCPResource `json:"resources"`
}

// ResourceReadParams are sent with the "resources/read" request.
type ResourceReadParams struct {
	URI string `json:"uri"`
}

// ResourceReadResult is the server's response to "resources/read".
type ResourceReadResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent is a single content block inside a resource-read result.
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}
