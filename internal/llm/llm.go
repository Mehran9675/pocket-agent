// Package llm is Pocket Agent's model-provider layer: one message
// format, one streaming interface, and adapters for the APIs people
// actually use - OpenAI-compatible endpoints (OpenAI, OpenRouter, Groq,
// Mistral, DeepSeek, xAI, Together, Ollama, LM Studio, llama.cpp
// server, ...), Anthropic and Google Gemini - plus the on-device engine
// in package local, which implements the same Provider interface.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// Role is the author of a message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// PartKind distinguishes message content parts.
type PartKind string

const (
	PartText  PartKind = "text"
	PartImage PartKind = "image"
)

// Part is one piece of message content.
type Part struct {
	Kind PartKind
	Text string
	// MIME and Data hold an image's encoded bytes (PNG, JPEG, WebP).
	MIME string
	Data []byte
}

// TextPart returns a text part.
func TextPart(s string) Part { return Part{Kind: PartText, Text: s} }

// ToolCall is a model's request to run a tool.
type ToolCall struct {
	ID   string
	Name string
	// Args is the JSON arguments object.
	Args json.RawMessage
}

// Message is one turn of a conversation in provider-neutral form.
type Message struct {
	Role  Role
	Parts []Part
	// ToolCalls are requested by an assistant message.
	ToolCalls []ToolCall
	// ToolCallID and ToolName identify the call a tool message answers.
	ToolCallID string
	ToolName   string
	// IsError marks a failed tool result.
	IsError bool
	// Native holds the provider's own encoding of an assistant message
	// (thinking blocks with signatures, say) for replay within the same
	// tool loop. It isn't persisted; Parts and ToolCalls stay canonical.
	Native any
}

// Text concatenates the message's text parts.
func (m Message) Text() string {
	var b strings.Builder
	for _, p := range m.Parts {
		if p.Kind == PartText {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

// ToolSpec describes a tool the model may call.
type ToolSpec struct {
	Name        string
	Description string
	// Schema is the JSON Schema of the arguments object.
	Schema map[string]any
}

// Request is one model call.
type Request struct {
	Model    string
	System   string
	Messages []Message
	Tools    []ToolSpec
	// Temperature is nil for the provider default.
	Temperature *float64
	// MaxTokens caps the reply; 0 uses a sensible default.
	MaxTokens int
	// Reasoning asks reasoning models to show their thinking, where
	// the API exposes a switch for it.
	Reasoning bool
	// Thinking sets how much reasoning models think.
	Thinking Thinking
}

// EventKind classifies streaming events.
type EventKind int

const (
	// EventText carries a piece of the reply.
	EventText EventKind = iota
	// EventReasoning carries a piece of the model's visible thinking.
	EventReasoning
	// EventToolCall reports a complete tool call.
	EventToolCall
)

// Event is one streaming update.
type Event struct {
	Kind     EventKind
	Text     string
	ToolCall ToolCall
}

// Usage counts tokens.
type Usage struct {
	InputTokens  int
	OutputTokens int
	// GenMillis is the time spent streaming the output, from the first
	// token on.
	GenMillis int64 `json:",omitempty"`
}

// Add sums two usages.
func (u Usage) Add(o Usage) Usage {
	return Usage{InputTokens: u.InputTokens + o.InputTokens, OutputTokens: u.OutputTokens + o.OutputTokens, GenMillis: u.GenMillis + o.GenMillis}
}

// TokensPerSecond is the output speed, 0 when unknown.
func (u Usage) TokensPerSecond() float64 {
	if u.GenMillis < 200 || u.OutputTokens == 0 {
		return 0
	}
	return float64(u.OutputTokens) / (float64(u.GenMillis) / 1000)
}

// Response is a finished model call.
type Response struct {
	Text      string
	Reasoning string
	ToolCalls []ToolCall
	Usage     Usage
	// StopReason is the provider's finish reason, normalized to "stop",
	// "length", "tool_calls", "refusal" or the raw value.
	StopReason string
	// Native is the provider's encoding of the reply; see Message.Native.
	Native any
}

// ModelInfo describes a model a provider offers.
type ModelInfo struct {
	ID   string
	Name string
	// ContextWindow in tokens, 0 when unknown.
	ContextWindow int
	Vision        bool
	Tools         bool
	Reasoning     bool
}

// Provider is a model backend.
type Provider interface {
	// Chat runs req, calling onEvent for every streaming update on the
	// calling goroutine, and returns the assembled response.
	Chat(ctx context.Context, req Request, onEvent func(Event)) (*Response, error)
	// Models lists the models the backend offers.
	Models(ctx context.Context) ([]ModelInfo, error)
}

// APIError is a non-2xx answer from a provider.
type APIError struct {
	Provider string
	Status   int
	Message  string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "request failed"
	}
	return e.Provider + ": " + msg
}

// ErrNoAPIKey is returned when a provider needs a key that isn't set.
var ErrNoAPIKey = errors.New("no API key configured for this provider")

// collector assembles a Response from events while forwarding them.
type collector struct {
	resp    Response
	onEvent func(Event)
	text    strings.Builder
	reason  strings.Builder
}

func newCollector(onEvent func(Event)) *collector {
	if onEvent == nil {
		onEvent = func(Event) {}
	}
	return &collector{onEvent: onEvent}
}

func (c *collector) textDelta(s string) {
	if s == "" {
		return
	}
	c.text.WriteString(s)
	c.onEvent(Event{Kind: EventText, Text: s})
}

func (c *collector) reasoningDelta(s string) {
	if s == "" {
		return
	}
	c.reason.WriteString(s)
	c.onEvent(Event{Kind: EventReasoning, Text: s})
}

func (c *collector) toolCall(tc ToolCall) {
	if len(tc.Args) == 0 || !json.Valid(tc.Args) {
		tc.Args = json.RawMessage("{}")
	}
	c.resp.ToolCalls = append(c.resp.ToolCalls, tc)
	c.onEvent(Event{Kind: EventToolCall, ToolCall: tc})
}

func (c *collector) finish() *Response {
	c.resp.Text = c.text.String()
	c.resp.Reasoning = c.reason.String()
	if c.resp.StopReason == "" {
		if len(c.resp.ToolCalls) > 0 {
			c.resp.StopReason = "tool_calls"
		} else {
			c.resp.StopReason = "stop"
		}
	}
	return &c.resp
}
