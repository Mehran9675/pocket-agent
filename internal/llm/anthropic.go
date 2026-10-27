package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Anthropic talks to the Claude Messages API through the official SDK.
type Anthropic struct {
	APIKey string
	// BaseURL overrides the API endpoint (a gateway or proxy).
	BaseURL string
}

func (p *Anthropic) client() anthropic.Client {
	opts := []option.RequestOption{
		// A phone app must not pick up a developer's environment.
		option.WithoutEnvironmentDefaults(),
		option.WithAPIKey(p.APIKey),
		option.WithHTTPClient(httpClient),
		option.WithMaxRetries(2),
	}
	if p.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(p.BaseURL))
	}
	return anthropic.NewClient(opts...)
}

// adaptiveThinking reports whether model takes thinking {type: "adaptive"}:
// the 4.6 generation onwards. Older models (Haiku 4.5, Sonnet 4.5, ...)
// are left on their defaults.
func adaptiveThinking(model string) bool {
	for _, old := range []string{"claude-3", "-4-5", "-4-1", "-4-0", "claude-opus-4-2", "claude-sonnet-4-2"} {
		if strings.Contains(model, old) {
			return false
		}
	}
	return strings.HasPrefix(model, "claude-")
}

// refusalFallback reports whether model accepts the server-side
// refusal fallback, which re-serves a declined request on a model the
// API picks by refusal category.
func refusalFallback(model string) bool {
	switch model {
	case "claude-fable-5-1", "claude-opus-5-5", "claude-opus-5", "claude-sonnet-5-5":
		return true
	}
	return false
}

func anthropicMessages(msgs []Message) []anthropic.MessageParam {
	var out []anthropic.MessageParam
	var pendingResults []anthropic.ContentBlockParamUnion
	flush := func() {
		if len(pendingResults) > 0 {
			// All results of one assistant turn go back in a single user
			// message.
			out = append(out, anthropic.NewUserMessage(pendingResults...))
			pendingResults = nil
		}
	}
	for _, m := range msgs {
		switch m.Role {
		case RoleTool:
			pendingResults = append(pendingResults, anthropic.NewToolResultBlock(m.ToolCallID, m.Text(), m.IsError))
		case RoleAssistant:
			flush()
			if native, ok := m.Native.(anthropic.MessageParam); ok {
				out = append(out, native)
				continue
			}
			var blocks []anthropic.ContentBlockParamUnion
			if t := m.Text(); t != "" {
				blocks = append(blocks, anthropic.NewTextBlock(t))
			}
			for _, tc := range m.ToolCalls {
				var input any = map[string]any{}
				_ = json.Unmarshal(tc.Args, &input)
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
			}
			if len(blocks) == 0 {
				blocks = append(blocks, anthropic.NewTextBlock("(no reply)"))
			}
			out = append(out, anthropic.NewAssistantMessage(blocks...))
		default:
			flush()
			var blocks []anthropic.ContentBlockParamUnion
			for _, part := range m.Parts {
				switch part.Kind {
				case PartImage:
					blocks = append(blocks, anthropic.NewImageBlockBase64(part.MIME, base64.StdEncoding.EncodeToString(part.Data)))
				case PartText:
					if part.Text != "" {
						blocks = append(blocks, anthropic.NewTextBlock(part.Text))
					}
				}
			}
			if len(blocks) == 0 {
				blocks = append(blocks, anthropic.NewTextBlock("."))
			}
			out = append(out, anthropic.NewUserMessage(blocks...))
		}
	}
	flush()
	return out
}

// Chat implements Provider.
func (p *Anthropic) Chat(ctx context.Context, req Request, onEvent func(Event)) (*Response, error) {
	if p.APIKey == "" {
		return nil, ErrNoAPIKey
	}
	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 32000
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: maxTokens,
		Messages:  anthropicMessages(req.Messages),
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}
	for _, t := range req.Tools {
		schema := anthropic.ToolInputSchemaParam{}
		if props, ok := t.Schema["properties"]; ok {
			schema.Properties = props
		}
		if reqd, ok := t.Schema["required"].([]string); ok {
			schema.Required = reqd
		}
		tool := anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: schema,
			// Large inputs (a file the agent writes) stream as they're
			// generated; the tool layer validates the JSON before running.
			EagerInputStreaming: anthropic.Bool(true),
		}
		params.Tools = append(params.Tools, anthropic.ToolUnionParam{OfTool: &tool})
	}
	if adaptiveThinking(req.Model) {
		if req.Thinking == ThinkingOff {
			params.Thinking = anthropic.ThinkingConfigParamUnion{OfDisabled: &anthropic.ThinkingConfigDisabledParam{}}
		} else {
			display := anthropic.ThinkingConfigAdaptiveDisplayOmitted
			if req.Reasoning {
				display = anthropic.ThinkingConfigAdaptiveDisplaySummarized
			}
			params.Thinking = anthropic.ThinkingConfigParamUnion{OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{Display: display}}
			if req.Thinking != ThinkingAuto {
				params.OutputConfig = anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffort(req.Thinking)}
			}
		}
	}
	var opts []option.RequestOption
	if refusalFallback(req.Model) {
		opts = append(opts,
			option.WithHeaderAdd("anthropic-beta", "server-side-fallback-2026-07-01"),
			option.WithJSONSet("fallbacks", "default"))
	}

	client := p.client()
	stream := client.Messages.NewStreaming(ctx, params, opts...)
	defer stream.Close()
	col := newCollector(onEvent)
	msg := anthropic.Message{}
	for stream.Next() {
		ev := stream.Current()
		if err := msg.Accumulate(ev); err != nil {
			return nil, err
		}
		if d, ok := ev.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
			switch delta := d.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				col.textDelta(delta.Text)
			case anthropic.ThinkingDelta:
				col.reasoningDelta(delta.Thinking)
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, anthropicError(err)
	}
	for _, block := range msg.Content {
		if tu, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
			col.toolCall(ToolCall{ID: tu.ID, Name: tu.Name, Args: json.RawMessage(tu.JSON.Input.Raw())})
		}
	}
	col.resp.Usage = Usage{InputTokens: int(msg.Usage.InputTokens), OutputTokens: int(msg.Usage.OutputTokens)}
	switch msg.StopReason {
	case anthropic.StopReasonToolUse:
		col.resp.StopReason = "tool_calls"
	case anthropic.StopReasonMaxTokens:
		col.resp.StopReason = "length"
	case anthropic.StopReasonRefusal:
		col.resp.StopReason = "refusal"
	default:
		col.resp.StopReason = "stop"
	}
	col.resp.Native = msg.ToParam()
	return col.finish(), nil
}

func anthropicError(err error) error {
	var apiErr *anthropic.Error
	if ae, ok := err.(*anthropic.Error); ok {
		apiErr = ae
	}
	if apiErr == nil {
		return err
	}
	msg := extractErrorMessage([]byte(apiErr.RawJSON()))
	if msg == "" {
		msg = apiErr.Error()
	}
	switch apiErr.StatusCode {
	case 401, 403:
		msg = "authentication failed - check the API key (" + msg + ")"
	case 429:
		msg = "rate limited - " + msg
	case 529:
		msg = "the API is overloaded - try again shortly"
	}
	return &APIError{Provider: "Anthropic", Status: apiErr.StatusCode, Message: msg}
}

// Models implements Provider.
func (p *Anthropic) Models(ctx context.Context) ([]ModelInfo, error) {
	if p.APIKey == "" {
		return nil, ErrNoAPIKey
	}
	client := p.client()
	pager := client.Models.ListAutoPaging(ctx, anthropic.ModelListParams{})
	var out []ModelInfo
	for pager.Next() {
		m := pager.Current()
		if m.Lifecycle == "retired" {
			continue
		}
		out = append(out, ModelInfo{
			ID:        m.ID,
			Name:      m.DisplayName,
			Vision:    true,
			Tools:     true,
			Reasoning: adaptiveThinking(m.ID),
		})
	}
	if err := pager.Err(); err != nil {
		return nil, anthropicError(err)
	}
	return out, nil
}
