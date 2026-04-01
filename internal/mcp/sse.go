package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	sseMaxRetries     = 3
	sseInitialBackoff = 1 * time.Second
	sseMaxBackoff     = 10 * time.Second
	sseEventBufSize   = 64
)

// SSETransport communicates with an MCP server over HTTP Server-Sent Events.
type SSETransport struct {
	baseURL    string
	httpClient *http.Client
	eventCh    chan *JSONRPCResponse
	sendURL    string
	closed     atomic.Bool
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	ready      chan struct{}
}

// NewSSETransport creates an SSE-based transport for the given endpoint URL.
func NewSSETransport(baseURL string) *SSETransport {
	return &SSETransport{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 0},
		eventCh:    make(chan *JSONRPCResponse, sseEventBufSize),
		ready:      make(chan struct{}),
	}
}

// Start opens the SSE connection and begins reading events.
func (t *SSETransport) Start(ctx context.Context) error {
	ctx, t.cancel = context.WithCancel(ctx)

	t.wg.Add(1)
	go t.readLoop(ctx)

	select {
	case <-t.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("SSE endpoint handshake timeout")
	}
}

func (t *SSETransport) readLoop(ctx context.Context) {
	defer t.wg.Done()
	defer close(t.eventCh)

	backoff := sseInitialBackoff
	for attempt := 0; attempt <= sseMaxRetries; attempt++ {
		if t.closed.Load() {
			return
		}

		err := t.connectAndRead(ctx)
		if t.closed.Load() || ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("mcp/sse: connection error (attempt %d/%d): %v", attempt+1, sseMaxRetries+1, err)
		}

		if attempt < sseMaxRetries {
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			backoff = min(backoff*2, sseMaxBackoff)
		}
	}
}

func (t *SSETransport) connectAndRead(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.baseURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SSE unexpected status: %d", resp.StatusCode)
	}

	return t.parseSSEStream(resp.Body)
}

func (t *SSETransport) parseSSEStream(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventType string
	var dataBuf bytes.Buffer

	for scanner.Scan() {
		if t.closed.Load() {
			return nil
		}

		line := scanner.Text()

		if line == "" {
			t.dispatchEvent(eventType, dataBuf.String())
			eventType = ""
			dataBuf.Reset()
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(strings.TrimPrefix(line, "data: "))
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("SSE read: %w", err)
	}
	return io.EOF
}

func (t *SSETransport) dispatchEvent(eventType, data string) {
	if data == "" {
		return
	}

	switch eventType {
	case "endpoint":
		t.handleEndpointEvent(data)
	default:
		t.handleMessageEvent(data)
	}
}

func (t *SSETransport) handleEndpointEvent(data string) {
	sendURL, err := resolveEndpointURL(t.baseURL, data)
	if err != nil {
		log.Printf("mcp/sse: invalid endpoint URL %q: %v", data, err)
		return
	}

	t.mu.Lock()
	t.sendURL = sendURL
	t.mu.Unlock()

	select {
	case <-t.ready:
	default:
		close(t.ready)
	}
}

func resolveEndpointURL(base, endpoint string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(endpointURL).String(), nil
}

func (t *SSETransport) handleMessageEvent(data string) {
	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		log.Printf("mcp/sse: invalid JSON-RPC response: %v", err)
		return
	}

	select {
	case t.eventCh <- &resp:
	default:
		log.Printf("mcp/sse: event channel full, dropping response id=%d", resp.ID)
	}
}

// Send posts a JSON-RPC request to the server's message endpoint.
func (t *SSETransport) Send(req *JSONRPCRequest) error {
	if t.closed.Load() {
		return fmt.Errorf("transport closed")
	}

	t.mu.Lock()
	sendURL := t.sendURL
	t.mu.Unlock()

	if sendURL == "" {
		return fmt.Errorf("SSE endpoint not yet established")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, sendURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create POST request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("POST %s: %w", sendURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("POST %s: status %d: %s", sendURL, resp.StatusCode, string(respBody))
	}
	return nil
}

// Receive blocks until the next JSON-RPC response arrives from the SSE stream.
func (t *SSETransport) Receive() (*JSONRPCResponse, error) {
	resp, ok := <-t.eventCh
	if !ok {
		return nil, io.EOF
	}
	return resp, nil
}

// Close shuts down the SSE connection.
func (t *SSETransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	if t.cancel != nil {
		t.cancel()
	}
	t.wg.Wait()
	return nil
}
