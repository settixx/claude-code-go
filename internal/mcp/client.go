package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/settixx/claude-code-go/internal/types"
)

const (
	protocolVersion = "2024-11-05"
	clientName      = "ti-code"
	clientVersion   = "0.1.0"
)

// Client manages a connection to a single MCP server.
type Client struct {
	name      string
	cfg       types.McpServerConfig
	transport Transport
	nextID    atomic.Int64
	mu        sync.Mutex
	connected bool
}

// NewClient creates a new MCP client for the named server.
func NewClient(name string, cfg types.McpServerConfig) *Client {
	return &Client{name: name, cfg: cfg}
}

// Name returns the server name this client is connected to.
func (c *Client) Name() string { return c.name }

// Connect starts the server process and performs the MCP initialize handshake.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connected {
		return nil
	}

	transport, err := c.buildTransport(ctx)
	if err != nil {
		return fmt.Errorf("build transport for %q: %w", c.name, err)
	}
	c.transport = transport

	if err := c.initialize(ctx); err != nil {
		_ = c.transport.Close()
		c.transport = nil
		return fmt.Errorf("initialize %q: %w", c.name, err)
	}
	c.connected = true
	return nil
}

func (c *Client) buildTransport(ctx context.Context) (Transport, error) {
	if c.cfg.URL != "" {
		st := NewSSETransport(c.cfg.URL)
		if err := st.Start(ctx); err != nil {
			return nil, fmt.Errorf("SSE start: %w", err)
		}
		return st, nil
	}
	env := buildEnv(c.cfg.Env)
	st := NewStdioTransport(c.cfg.Command, c.cfg.Args, env)
	if err := st.Start(ctx); err != nil {
		return nil, err
	}
	return st, nil
}

func buildEnv(extra map[string]string) []string {
	if len(extra) == 0 {
		return nil
	}
	env := os.Environ()
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}

func (c *Client) initialize(_ context.Context) error {
	params := InitializeParams{
		ProtocolVersion: protocolVersion,
		ClientInfo:      ClientInfo{Name: clientName, Version: clientVersion},
		Capabilities: ClientCapabilities{
			Tools:     &ToolCapabilities{},
			Resources: &ResourceCapabilities{},
		},
	}
	_, err := c.call(MethodInitialize, params)
	return err
}

// Disconnect sends shutdown and closes the transport.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return nil
	}
	_, _ = c.call(MethodShutdown, nil)
	err := c.transport.Close()
	c.connected = false
	c.transport = nil
	return err
}

// ListTools retrieves all tools exposed by the server.
func (c *Client) ListTools(ctx context.Context) ([]MCPToolSchema, error) {
	raw, err := c.roundTrip(ctx, MethodToolsList, nil)
	if err != nil {
		return nil, err
	}
	return decodeResult[ToolsListResult](raw, func(r ToolsListResult) []MCPToolSchema { return r.Tools })
}

// CallTool invokes a tool on the server and returns the result content.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolCallResult, error) {
	params := ToolCallParams{Name: name, Arguments: args}
	raw, err := c.roundTrip(ctx, MethodToolsCall, params)
	if err != nil {
		return nil, err
	}
	var result ToolCallResult
	if err := remarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode tool call result: %w", err)
	}
	return &result, nil
}

// ListResources retrieves all resources exposed by the server.
func (c *Client) ListResources(ctx context.Context) ([]MCPResource, error) {
	raw, err := c.roundTrip(ctx, MethodResourceList, nil)
	if err != nil {
		return nil, err
	}
	return decodeResult[ResourceListResult](raw, func(r ResourceListResult) []MCPResource { return r.Resources })
}

// ReadResource fetches the content of a resource by URI.
func (c *Client) ReadResource(ctx context.Context, uri string) (string, error) {
	params := ResourceReadParams{URI: uri}
	raw, err := c.roundTrip(ctx, MethodResourceRead, params)
	if err != nil {
		return "", err
	}
	var result ResourceReadResult
	if err := remarshal(raw, &result); err != nil {
		return "", fmt.Errorf("decode resource read result: %w", err)
	}
	if len(result.Contents) == 0 {
		return "", nil
	}
	return result.Contents[0].Text, nil
}

func (c *Client) roundTrip(_ context.Context, method string, params interface{}) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected && method != MethodInitialize {
		return nil, fmt.Errorf("client %q not connected", c.name)
	}
	return c.call(method, params)
}

func (c *Client) call(method string, params interface{}) (interface{}, error) {
	id := c.nextID.Add(1)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	if err := c.transport.Send(req); err != nil {
		return nil, fmt.Errorf("send %s: %w", method, err)
	}
	resp, err := c.transport.Receive()
	if err != nil {
		return nil, fmt.Errorf("receive %s: %w", method, err)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Result, nil
}

// decodeResult re-marshals a raw interface{} into T, then extracts a slice via fn.
func decodeResult[T any, S any](raw interface{}, fn func(T) S) (S, error) {
	var result T
	if err := remarshal(raw, &result); err != nil {
		var zero S
		return zero, fmt.Errorf("decode result: %w", err)
	}
	return fn(result), nil
}

// remarshal round-trips through JSON to convert interface{} → concrete struct.
func remarshal(src interface{}, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
