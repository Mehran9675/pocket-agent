package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
)

// OpenAI talks to any OpenAI-compatible Chat Completions endpoint.
type OpenAI struct {
	// Name labels errors ("OpenAI", "OpenRouter", "Ollama", ...).
	Name    string
	BaseURL string
	APIKey  string
	// Headers are sent with every request (OpenRouter's attribution
	// headers, for instance).
	Headers map[string]string
	// KeyOptional is set for local servers that need no key.
	KeyOptional bool
	// Official marks api.openai.com, which wants
	// max_completion_tokens and supports stream usage.
	Official bool
	// NoStreamUsage leaves out stream_options for servers that reject
	// it.
	NoStreamUsage bool
}

func (p *OpenAI) headers() map[string]string {
	h := map[string]string{}
	for k, v := range p.Headers {
		h[k] = v
	}
	if p.APIKey != "" {
		h["Authorization"] = "Bearer " + p.APIKey
	}
	return h
}

func (p *OpenAI) base() string { return strings.TrimRight(p.BaseURL, "/") }

type oaMessage struct {
	Role       string       `json:"role"`
	Content    any          `json:"content"`
	ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
	Name       string       `json:"name,omitempty"`
}

type oaToolCall struct {
	Index    *int   `json:"index,omitempty"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
}

func oaMessages(system string, msgs []Message) []oaMessage {
	var out []oaMessage
	if system != "" {
		out = append(out, oaMessage{Role: "system", Content: system})
	}
	for _, m := range msgs {
		switch m.Role {
		case RoleTool:
			out = append(out, oaMessage{Role: "tool", Content: m.Text(), ToolCallID: m.ToolCallID})
		case RoleAssistant:
			om := oaMessage{Role: "assistant", Content: m.Text()}
			for _, tc := range m.ToolCalls {
				var c oaToolCall
				c.ID, c.Type = tc.ID, "function"
				c.Function.Name, c.Function.Arguments = tc.Name, string(tc.Args)
				om.ToolCalls = append(om.ToolCalls, c)
			}
			if len(om.ToolCalls) > 0 && m.Text() == "" {
				om.Content = nil
			}
			out = append(out, om)
		default:
			role := string(m.Role)
			hasImage := false
			for _, part := range m.Parts {
				if part.Kind == PartImage {
					hasImage = true
				}
			}
			if !hasImage {
				out = append(out, oaMessage{Role: role, Content: m.Text()})
				continue
			}
			var parts []oaContentPart
			for _, part := range m.Parts {
				switch part.Kind {
				case PartText:
					parts = append(parts, oaContentPart{Type: "text", Text: part.Text})
				case PartImage:
					cp := oaContentPart{Type: "image_url"}
					cp.ImageURL = &struct {
						URL string `json:"url"`
					}{URL: "data:" + part.MIME + ";base64," + base64.StdEncoding.EncodeToString(part.Data)}
					parts = append(parts, cp)
				}
			}
			out = append(out, oaMessage{Role: role, Content: parts})
		}
	}
	return out
}

func oaTools(tools []ToolSpec) []map[string]any {
	var out []map[string]any
	for _, t := range tools {
		schema := t.Schema
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  schema,
			},
		})
	}
	return out
}

// Chat implements Provider.
func (p *OpenAI) Chat(ctx context.Context, req Request, onEvent func(Event)) (*Response, error) {
	if p.APIKey == "" && !p.KeyOptional {
		return nil, ErrNoAPIKey
	}
	body := map[string]any{
		"model":    req.Model,
		"messages": oaMessages(req.System, req.Messages),
		"stream":   true,
	}
	if !p.NoStreamUsage {
		body["stream_options"] = map[string]any{"include_usage": true}
	}
	if len(req.Tools) > 0 {
		body["tools"] = oaTools(req.Tools)
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.MaxTokens > 0 {
		if p.Official {
			body["max_completion_tokens"] = req.MaxTokens
		} else {
			body["max_tokens"] = req.MaxTokens
		}
	}
	if req.Reasoning && p.Official {
		body["reasoning_effort"] = "medium"
	}
	resp, err := postJSON(ctx, p.Name, p.base()+"/chat/completions", p.headers(), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	col := newCollector(onEvent)
	var think ThinkSplitter
	calls := map[int]*ToolCall{}
	args := map[int]*strings.Builder{}
	err = readSSE(resp.Body, func(_, data string) error {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string       `json:"content"`
					ReasoningContent string       `json:"reasoning_content"`
					Reasoning        string       `json:"reasoning"`
					ToolCalls        []oaToolCall `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil // tolerate keep-alive junk
		}
		if chunk.Error != nil {
			return &APIError{Provider: p.Name, Message: chunk.Error.Message}
		}
		if chunk.Usage != nil {
			col.resp.Usage = Usage{InputTokens: chunk.Usage.PromptTokens, OutputTokens: chunk.Usage.CompletionTokens}
		}
		for _, ch := range chunk.Choices {
			d := ch.Delta
			col.reasoningDelta(d.ReasoningContent)
			col.reasoningDelta(d.Reasoning)
			if d.Content != "" {
				t, r := think.Feed(d.Content)
				col.reasoningDelta(r)
				col.textDelta(t)
			}
			for i, tc := range d.ToolCalls {
				idx := i
				if tc.Index != nil {
					idx = *tc.Index
				}
				c, ok := calls[idx]
				if !ok {
					c = &ToolCall{}
					calls[idx] = c
					args[idx] = &strings.Builder{}
				}
				if tc.ID != "" {
					c.ID = tc.ID
				}
				if tc.Function.Name != "" {
					c.Name += tc.Function.Name
				}
				args[idx].WriteString(tc.Function.Arguments)
			}
			if ch.FinishReason != "" {
				col.resp.StopReason = normalizeFinish(ch.FinishReason)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	t, r := think.Flush()
	col.reasoningDelta(r)
	col.textDelta(t)
	idxs := make([]int, 0, len(calls))
	for i := range calls {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	for _, i := range idxs {
		c := calls[i]
		c.Args = json.RawMessage(args[i].String())
		if c.ID == "" {
			c.ID = newCallID()
		}
		col.toolCall(*c)
	}
	return col.finish(), nil
}

func normalizeFinish(r string) string {
	switch r {
	case "stop", "end_turn", "STOP", "stop_sequence":
		return "stop"
	case "length", "max_tokens", "MAX_TOKENS":
		return "length"
	case "tool_calls", "tool_use", "function_call":
		return "tool_calls"
	}
	return r
}

// Models implements Provider.
func (p *OpenAI) Models(ctx context.Context) ([]ModelInfo, error) {
	var out struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ContextLength int    `json:"context_length"`
		} `json:"data"`
	}
	if err := getJSON(ctx, p.Name, p.base()+"/models", p.headers(), &out); err != nil {
		return nil, err
	}
	var models []ModelInfo
	for _, m := range out.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		models = append(models, ModelInfo{ID: m.ID, Name: name, ContextWindow: m.ContextLength, Tools: true})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}
