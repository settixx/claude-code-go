package types

// QuerySource identifies where a query request originated.
type QuerySource string

const (
	QueryREPL      QuerySource = "repl"
	QueryPrint     QuerySource = "print"
	QuerySDK       QuerySource = "sdk"
	QuerySubagent  QuerySource = "subagent"
)

// QueryConfig parameterizes a single query to the LLM.
type QueryConfig struct {
	Model           string      `json:"model"`
	MaxTokens       int         `json:"max_tokens,omitempty"`
	Temperature     float64     `json:"temperature,omitempty"`
	TopP            float64     `json:"top_p,omitempty"`
	StopSequences   []string    `json:"stop_sequences,omitempty"`
	SystemPrompt    string      `json:"system,omitempty"`
	Tools           []ToolDef   `json:"tools,omitempty"`
	ThinkingBudget  int         `json:"thinking_budget,omitempty"`
	MaxBudgetUSD    float64     `json:"max_budget_usd,omitempty"`
	Source          QuerySource `json:"source,omitempty"`
}

// ToolDef is the API-wire representation of a tool schema.
type ToolDef struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  ToolInputSchema `json:"input_schema"`
}

// StreamEventType classifies SSE events from the API.
type StreamEventType string

const (
	EventMessageStart      StreamEventType = "message_start"
	EventContentBlockStart StreamEventType = "content_block_start"
	EventContentBlockDelta StreamEventType = "content_block_delta"
	EventContentBlockStop  StreamEventType = "content_block_stop"
	EventMessageDelta      StreamEventType = "message_delta"
	EventMessageStop       StreamEventType = "message_stop"
	EventPing              StreamEventType = "ping"
	EventError             StreamEventType = "error"
)

// StreamEvent wraps one SSE event from the Anthropic streaming API.
type StreamEvent struct {
	Type    StreamEventType `json:"type"`
	Index   int             `json:"index,omitempty"`
	Message *APIMessage     `json:"message,omitempty"`

	ContentBlock *ContentBlock `json:"content_block,omitempty"`
	Delta        *DeltaBlock   `json:"delta,omitempty"`

	Usage *Usage `json:"usage,omitempty"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// DeltaBlock carries partial content from a streaming delta event.
type DeltaBlock struct {
	Type         string     `json:"type"`
	Text         string     `json:"text,omitempty"`
	PartialJSON  string     `json:"partial_json,omitempty"`
	Thinking     string     `json:"thinking,omitempty"`
	StopReason   StopReason `json:"stop_reason,omitempty"`
}
