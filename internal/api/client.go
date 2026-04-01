package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	apierrors "github.com/settixx/claude-code-go/internal/errors"
	"github.com/settixx/claude-code-go/internal/types"
)

const (
	defaultBaseURL    = "https://api.anthropic.com"
	defaultTimeout    = 10 * time.Minute
	messagesEndpoint  = "/v1/messages"
	countTokensPath   = "/v1/messages/count_tokens"
	anthropicVersion  = "2023-06-01"
)

// ClientConfig holds the parameters needed to construct a Client.
type ClientConfig struct {
	APIKey       string
	BaseURL      string
	DefaultModel string
	MaxRetries   int
	Timeout      time.Duration
	HTTPClient   *http.Client
}

func (cc ClientConfig) baseURL() string {
	if cc.BaseURL != "" {
		return cc.BaseURL
	}
	return defaultBaseURL
}

func (cc ClientConfig) timeout() time.Duration {
	if cc.Timeout > 0 {
		return cc.Timeout
	}
	return defaultTimeout
}

func (cc ClientConfig) httpClient() *http.Client {
	if cc.HTTPClient != nil {
		return cc.HTTPClient
	}
	return &http.Client{Timeout: cc.timeout()}
}

// Client communicates with the Anthropic Messages API. It implements
// the interfaces.LLMClient interface (Stream, Send, CountTokens).
type Client struct {
	apiKey       string
	baseURL      string
	defaultModel string
	httpClient   *http.Client
	retryCfg     RetryConfig
	costTracker  *CostTracker
}

// NewClient constructs a Client from the given config.
func NewClient(cfg ClientConfig) *Client {
	return &Client{
		apiKey:       cfg.APIKey,
		baseURL:      cfg.baseURL(),
		defaultModel: cfg.DefaultModel,
		httpClient:   cfg.httpClient(),
		retryCfg: RetryConfig{
			MaxRetries: cfg.MaxRetries,
		},
		costTracker: NewCostTracker(),
	}
}

// CostTracker returns the client's accumulated cost tracker.
func (c *Client) CostTracker() *CostTracker { return c.costTracker }

// --- LLMClient interface ---

// Stream sends a streaming request and returns a channel of StreamEvents.
// The channel is closed when the stream ends or the context is cancelled.
func (c *Client) Stream(ctx context.Context, config types.QueryConfig, messages []types.Message) (<-chan types.StreamEvent, error) {
	body, err := c.buildRequestBody(config, messages, true)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequestWithRetry(ctx, messagesEndpoint, body)
	if err != nil {
		return nil, err
	}

	raw := ReadSSEStream(resp.Body)

	out := make(chan types.StreamEvent, 16)
	go c.trackStreamCost(config, raw, out)
	return out, nil
}

// Send performs a non-streaming request and returns the complete API response.
func (c *Client) Send(ctx context.Context, config types.QueryConfig, messages []types.Message) (*types.APIMessage, error) {
	body, err := c.buildRequestBody(config, messages, false)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequestWithRetry(ctx, messagesEndpoint, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var msg types.APIMessage
	if decodeErr := json.NewDecoder(resp.Body).Decode(&msg); decodeErr != nil {
		return nil, fmt.Errorf("decode API response: %w", decodeErr)
	}

	if msg.Usage != nil {
		model := c.resolveModel(config.Model)
		c.costTracker.Add(model, *msg.Usage)
	}
	return &msg, nil
}

// CountTokens estimates the token count for a set of messages by calling
// the token-counting endpoint.
func (c *Client) CountTokens(ctx context.Context, config types.QueryConfig, messages []types.Message) (int, error) {
	body, err := c.buildCountTokensBody(config, messages)
	if err != nil {
		return 0, err
	}

	resp, err := c.doRequestWithRetry(ctx, countTokensPath, body)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		InputTokens int `json:"input_tokens"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		return 0, fmt.Errorf("decode token count response: %w", decodeErr)
	}
	return result.InputTokens, nil
}

// --- internal helpers ---

// trackStreamCost forwards events from the raw SSE channel to the output
// channel while accumulating usage data for cost tracking.
func (c *Client) trackStreamCost(config types.QueryConfig, raw <-chan types.StreamEvent, out chan<- types.StreamEvent) {
	defer close(out)
	model := c.resolveModel(config.Model)
	var lastUsage *types.Usage

	for evt := range raw {
		if evt.Usage != nil {
			lastUsage = evt.Usage
		}
		if evt.Type == types.EventMessageStart && evt.Message != nil && evt.Message.Usage != nil {
			lastUsage = evt.Message.Usage
		}
		if evt.Type == types.EventMessageDelta && evt.Usage != nil {
			lastUsage = evt.Usage
		}
		out <- evt
	}

	if lastUsage != nil {
		c.costTracker.Add(model, *lastUsage)
	}
}

func (c *Client) resolveModel(model string) string {
	if model != "" {
		return ResolveModel(model)
	}
	return ResolveModel(c.defaultModel)
}

// apiRequest is the JSON body sent to the Messages API.
type apiRequest struct {
	Model         string              `json:"model"`
	MaxTokens     int                 `json:"max_tokens"`
	Messages      []apiMessageParam   `json:"messages"`
	System        interface{}         `json:"system,omitempty"`
	Stream        bool                `json:"stream"`
	Temperature   *float64            `json:"temperature,omitempty"`
	TopP          *float64            `json:"top_p,omitempty"`
	StopSequences []string            `json:"stop_sequences,omitempty"`
	Tools         []types.ToolDef     `json:"tools,omitempty"`
	Metadata      *apiRequestMetadata `json:"metadata,omitempty"`
	Thinking      *ThinkingConfig     `json:"thinking,omitempty"`
}

// ThinkingConfig controls extended thinking (chain-of-thought) for models that support it.
type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

type apiRequestMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// SystemBlock is a structured system prompt fragment with optional cache control.
type SystemBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// CacheControl instructs the API to cache the associated block.
type CacheControl struct {
	Type string `json:"type"`
}

type apiMessageParam struct {
	Role    string               `json:"role"`
	Content []types.ContentBlock `json:"content"`
}

func (c *Client) buildRequestBody(config types.QueryConfig, messages []types.Message, stream bool) ([]byte, error) {
	model := c.resolveModel(config.Model)
	maxTokens := config.MaxTokens
	if maxTokens <= 0 {
		caps := GetModelCapabilities(model)
		maxTokens = caps.MaxOutputTokens
	}

	apiMsgs := convertMessages(messages)

	req := apiRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  apiMsgs,
		System:    buildSystemBlocks(config.SystemPrompt),
		Stream:    stream,
	}

	caps := GetModelCapabilities(model)
	thinkingEnabled := config.ThinkingBudget > 0 && caps.SupportsThinking
	if thinkingEnabled {
		req.Thinking = &ThinkingConfig{Type: "enabled", BudgetTokens: config.ThinkingBudget}
		t := 1.0
		req.Temperature = &t
		req.TopP = nil
	} else {
		if config.Temperature > 0 {
			t := config.Temperature
			req.Temperature = &t
		}
		if config.TopP > 0 {
			p := config.TopP
			req.TopP = &p
		}
	}

	if len(config.StopSequences) > 0 {
		req.StopSequences = config.StopSequences
	}
	if len(config.Tools) > 0 {
		req.Tools = config.Tools
	}

	return json.Marshal(req)
}

func (c *Client) buildCountTokensBody(config types.QueryConfig, messages []types.Message) ([]byte, error) {
	model := c.resolveModel(config.Model)
	apiMsgs := convertMessages(messages)

	payload := struct {
		Model    string            `json:"model"`
		Messages []apiMessageParam `json:"messages"`
		System   interface{}       `json:"system,omitempty"`
		Tools    []types.ToolDef   `json:"tools,omitempty"`
	}{
		Model:    model,
		Messages: apiMsgs,
		System:   buildSystemBlocks(config.SystemPrompt),
	}
	if len(config.Tools) > 0 {
		payload.Tools = config.Tools
	}

	return json.Marshal(payload)
}

// buildSystemBlocks splits a system prompt at the dynamic boundary marker.
// The static part before the boundary gets cache_control for prompt caching;
// the dynamic part after does not. If no boundary is present the plain string is returned.
func buildSystemBlocks(systemPrompt string) interface{} {
	if systemPrompt == "" {
		return nil
	}

	const boundary = "SYSTEM_PROMPT_DYNAMIC_BOUNDARY"
	idx := strings.Index(systemPrompt, boundary)
	if idx < 0 {
		return systemPrompt
	}

	staticPart := strings.TrimSpace(systemPrompt[:idx])
	dynamicPart := strings.TrimSpace(systemPrompt[idx+len(boundary):])

	blocks := []SystemBlock{
		{Type: "text", Text: staticPart, CacheControl: &CacheControl{Type: "ephemeral"}},
	}
	if dynamicPart != "" {
		blocks = append(blocks, SystemBlock{Type: "text", Text: dynamicPart})
	}
	return blocks
}

// convertMessages transforms the high-level Message slice into the
// role/content pairs expected by the API, filtering out non-API messages.
func convertMessages(messages []types.Message) []apiMessageParam {
	result := make([]apiMessageParam, 0, len(messages))

	for _, msg := range messages {
		switch msg.Type {
		case types.MsgUser:
			content := resolveContent(msg)
			if len(content) > 0 {
				result = append(result, apiMessageParam{Role: "user", Content: content})
			}
		case types.MsgAssistant:
			if msg.APIMessage != nil && len(msg.APIMessage.Content) > 0 {
				result = append(result, apiMessageParam{
					Role:    "assistant",
					Content: msg.APIMessage.Content,
				})
			}
		}
	}

	return result
}

func resolveContent(msg types.Message) []types.ContentBlock {
	if len(msg.Content) > 0 {
		return msg.Content
	}
	if msg.Text != "" {
		return []types.ContentBlock{{Type: types.ContentText, Text: msg.Text}}
	}
	return nil
}

func (c *Client) doRequestWithRetry(ctx context.Context, path string, body []byte) (*http.Response, error) {
	return WithRetry(ctx, c.retryCfg, func(ctx context.Context) (*http.Response, error) {
		return c.doRequest(ctx, path, body)
	})
}

func (c *Client) doRequest(ctx context.Context, path string, body []byte) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Anthropic-Version", anthropicVersion)
	req.Header.Set("Accept", "application/json")

	if bytes.Contains(body, []byte(`"thinking"`)) {
		req.Header.Set("Anthropic-Beta", "interleaved-thinking-2025-05-14")
	}

	slog.Debug("API request", "method", "POST", "url", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &apierrors.APIError{
			StatusCode: 0,
			Type:       "connection_error",
			Message:    err.Error(),
			Retryable:  true,
		}
	}

	if resp.StatusCode >= 400 {
		return resp, ParseAPIErrorResponse(resp)
	}

	return resp, nil
}
