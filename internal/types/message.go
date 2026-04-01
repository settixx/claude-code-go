package types

import "time"

// MessageOriginKind distinguishes how a message entered the conversation.
type MessageOriginKind string

const (
	OriginHuman            MessageOriginKind = "human"
	OriginTaskNotification MessageOriginKind = "task-notification"
	OriginCoordinator      MessageOriginKind = "coordinator"
	OriginChannel          MessageOriginKind = "channel"
)

type MessageOrigin struct {
	Kind   MessageOriginKind `json:"kind"`
	Server string            `json:"server,omitempty"`
}

// SystemMessageLevel controls the severity shown in the TUI.
type SystemMessageLevel string

const (
	LevelInfo    SystemMessageLevel = "info"
	LevelWarning SystemMessageLevel = "warning"
	LevelError   SystemMessageLevel = "error"
)

// PartialCompactDirection indicates which direction a partial compact applies.
type PartialCompactDirection string

const (
	CompactFrom PartialCompactDirection = "from"
	CompactUpTo PartialCompactDirection = "up_to"
)

// ContentBlockType distinguishes different content block kinds.
type ContentBlockType string

const (
	ContentText      ContentBlockType = "text"
	ContentToolUse   ContentBlockType = "tool_use"
	ContentToolResult ContentBlockType = "tool_result"
	ContentImage     ContentBlockType = "image"
	ContentThinking  ContentBlockType = "thinking"
)

// ContentBlock is a single block within a message's content array.
type ContentBlock struct {
	Type  ContentBlockType       `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`

	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   []ContentBlock `json:"content,omitempty"`
	IsError   bool           `json:"is_error,omitempty"`

	Source    *ImageSource `json:"source,omitempty"`
	Thinking  string       `json:"thinking,omitempty"`
}

// ImageSource holds base64-encoded image data.
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// StopReason indicates why the model stopped generating.
type StopReason string

const (
	StopEndTurn   StopReason = "end_turn"
	StopMaxTokens StopReason = "max_tokens"
	StopToolUse   StopReason = "tool_use"
)

// APIMessage is the wire format exchanged with the LLM API.
type APIMessage struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model,omitempty"`
	StopReason   StopReason     `json:"stop_reason,omitempty"`
	StopSequence *string        `json:"stop_sequence,omitempty"`
	Usage        *Usage         `json:"usage,omitempty"`
}

// Usage tracks token consumption for a single API response.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// MessageType distinguishes the top-level message variants.
type MessageType string

const (
	MsgUser       MessageType = "user"
	MsgAssistant  MessageType = "assistant"
	MsgSystem     MessageType = "system"
	MsgAttachment MessageType = "attachment"
	MsgProgress   MessageType = "progress"
)

// Message is the unified envelope for all conversation messages.
type Message struct {
	Type      MessageType `json:"type"`
	UUID      string      `json:"uuid"`
	Timestamp time.Time   `json:"timestamp"`

	// User message fields
	Role    string         `json:"role,omitempty"`
	Content []ContentBlock `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`

	IsMeta                    bool                    `json:"is_meta,omitempty"`
	IsVisibleInTranscriptOnly bool                    `json:"is_visible_in_transcript_only,omitempty"`
	IsVirtual                 bool                    `json:"is_virtual,omitempty"`
	IsCompactSummary          bool                    `json:"is_compact_summary,omitempty"`
	SummarizeMetadata         *SummarizeMetadata      `json:"summarize_metadata,omitempty"`
	PermissionMode            PermissionMode          `json:"permission_mode,omitempty"`
	Origin                    *MessageOrigin          `json:"origin,omitempty"`

	// Assistant message fields
	APIMessage *APIMessage `json:"api_message,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
	IsAPIError bool        `json:"is_api_error,omitempty"`
	APIError   string      `json:"api_error,omitempty"`

	// System message fields
	Subtype SystemMessageSubtype   `json:"subtype,omitempty"`
	Level   SystemMessageLevel     `json:"level,omitempty"`

	// Progress message fields
	ToolUseID       string      `json:"tool_use_id,omitempty"`
	ParentToolUseID string      `json:"parent_tool_use_id,omitempty"`
	ProgressData    interface{} `json:"progress_data,omitempty"`
}

// SummarizeMetadata holds compaction context.
type SummarizeMetadata struct {
	MessagesSummarized int                     `json:"messages_summarized"`
	UserContext        string                  `json:"user_context,omitempty"`
	Direction          PartialCompactDirection `json:"direction,omitempty"`
}

// SystemMessageSubtype distinguishes system message variants.
type SystemMessageSubtype string

const (
	SubtypeInformational      SystemMessageSubtype = "informational"
	SubtypeAPIError           SystemMessageSubtype = "api_error"
	SubtypeCompactBoundary    SystemMessageSubtype = "compact_boundary"
	SubtypeLocalCommand       SystemMessageSubtype = "local_command"
	SubtypeBridgeStatus       SystemMessageSubtype = "bridge_status"
	SubtypePermissionRetry    SystemMessageSubtype = "permission_retry"
	SubtypeScheduledTaskFire  SystemMessageSubtype = "scheduled_task_fire"
	SubtypeStopHookSummary    SystemMessageSubtype = "stop_hook_summary"
	SubtypeTurnDuration       SystemMessageSubtype = "turn_duration"
	SubtypeAwaySummary        SystemMessageSubtype = "away_summary"
	SubtypeMemorySaved        SystemMessageSubtype = "memory_saved"
	SubtypeAgentsKilled       SystemMessageSubtype = "agents_killed"
	SubtypeAPIMetrics         SystemMessageSubtype = "api_metrics"
)

// CompactMetadata describes a compaction event.
type CompactMetadata struct {
	Trigger                   string            `json:"trigger"`
	PreTokens                 int               `json:"pre_tokens"`
	UserContext               string            `json:"user_context,omitempty"`
	MessagesSummarized        int               `json:"messages_summarized,omitempty"`
	PreCompactDiscoveredTools []string           `json:"pre_compact_discovered_tools,omitempty"`
	PreservedSegment          *PreservedSegment  `json:"preserved_segment,omitempty"`
}

type PreservedSegment struct {
	HeadUUID   string `json:"head_uuid"`
	AnchorUUID string `json:"anchor_uuid"`
	TailUUID   string `json:"tail_uuid"`
}

// StopHookInfo describes the outcome of a single stop hook.
type StopHookInfo struct {
	HookName   string `json:"hook_name"`
	Outcome    string `json:"outcome"`
	DurationMs int    `json:"duration_ms,omitempty"`
}
