package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Transport abstracts the communication channel to an MCP server.
type Transport interface {
	// Send writes a JSON-RPC request to the server.
	Send(req *JSONRPCRequest) error
	// Receive blocks until the next JSON-RPC response arrives.
	Receive() (*JSONRPCResponse, error)
	// Close shuts down the transport and releases resources.
	Close() error
}

// StdioTransport spawns a child process and communicates via line-delimited
// JSON-RPC on its stdin/stdout.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex
	closed atomic.Bool
}

// NewStdioTransport creates a transport that will spawn the given command.
// The process is not started until Start is called.
func NewStdioTransport(command string, args []string, env []string) *StdioTransport {
	cmd := exec.Command(command, args...)
	if len(env) > 0 {
		cmd.Env = env
	}
	return &StdioTransport{cmd: cmd}
}

// Start launches the child process and wires up stdin/stdout pipes.
func (t *StdioTransport) Start(ctx context.Context) error {
	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdoutPipe, err := t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	t.stdout = bufio.NewScanner(stdoutPipe)
	t.stdout.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	go func() {
		<-ctx.Done()
		_ = t.Close()
	}()

	return nil
}

// Send writes a JSON-RPC request as a single line to the child's stdin.
func (t *StdioTransport) Send(req *JSONRPCRequest) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed.Load() {
		return fmt.Errorf("transport closed")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("write stdin: %w", err)
	}
	return nil
}

// Receive reads the next line-delimited JSON-RPC response from stdout.
func (t *StdioTransport) Receive() (*JSONRPCResponse, error) {
	if !t.stdout.Scan() {
		if err := t.stdout.Err(); err != nil {
			return nil, fmt.Errorf("read stdout: %w", err)
		}
		return nil, io.EOF
	}
	var resp JSONRPCResponse
	if err := json.Unmarshal(t.stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

// Close shuts down the child process.
func (t *StdioTransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	_ = t.stdin.Close()
	return t.cmd.Process.Kill()
}
