package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

// Gemini talks to Google's Generative Language API.
type Gemini struct {
	APIKey  string
	BaseURL string // defaults to https://generativelanguage.googleapis.com/v1beta
}

func (p *Gemini) base() string {
	if p.BaseURL != "" {
		return strings.TrimRight(p.BaseURL, "/")
	}
	return "https://generativelanguage.googleapis.com/v1beta"
}

type gmPart struct {
	Text       string `json:"text,omitempty"`
	Thought    bool   `json:"thought,omitempty"`
	InlineData *struct {
		MimeType string `json:"mimeType"`
		Data     string `json:"data"`
	} `json:"inlineData,omitempty"`
	FunctionCall *struct {
		Name string          `json:"name"`
		Args json.RawMessage `json:"args,omitempty"`
		ID   string          `json:"id,omitempty"`
	} `json:"functionCall,omitempty"`
	FunctionResponse *struct {
		Name     string         `json:"name"`
		Response map[string]any `json:"response"`
		ID       string         `json:"id,omitempty"`
	} `json:"functionResponse,omitempty"`
	ThoughtSignature string `json:"thoughtSignature,omitempty"`
}

type gmContent struct {
	Role  string   `json:"role,omitempty"`
	Parts []gmPart `json:"parts"`
}

func geminiContents(msgs []Message) []gmContent {
	var out []gmContent
	add := func(role string, parts ...gmPart) {
		if n := len(out); n > 0 && out[n-1].Role == role {
			out[n-1].Parts = append(out[n-1].Parts, parts...)
			return
		}
		out = append(out, gmContent{Role: role, Parts: parts})
	}
	for _, m := range msgs {
		switch m.Role {
		case RoleTool:
			var p gmPart
			p.FunctionResponse = &struct {
				Name     string         `json:"name"`
				Response map[string]any `json:"response"`
				ID       string         `json:"id,omitempty"`
			}{Name: m.ToolName, Response: map[string]any{"content": m.Text(), "is_error": m.IsError}}
			add("user", p)
		case RoleAssistant:
			if native, ok := m.Native.([]gmPart); ok {
				add("model", native...)
				continue
			}
			var parts []gmPart
			if t := m.Text(); t != "" {
				parts = append(parts, gmPart{Text: t})
			}
			for _, tc := range m.ToolCalls {
				var p gmPart
				p.FunctionCall = &struct {
					Name string          `json:"name"`
					Args json.RawMessage `json:"args,omitempty"`
					ID   string          `json:"id,omitempty"`
				}{Name: tc.Name, Args: tc.Args}
				parts = append(parts, p)
			}
			if len(parts) == 0 {
				parts = append(parts, gmPart{Text: " "})
			}
			add("model", parts...)
		default:
			var parts []gmPart
			for _, part := range m.Parts {
				switch part.Kind {
				case PartImage:
					var p gmPart
					p.InlineData = &struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					}{part.MIME, base64.StdEncoding.EncodeToString(part.Data)}
					parts = append(parts, p)
				case PartText:
					if part.Text != "" {
						parts = append(parts, gmPart{Text: part.Text})
					}
				}
			}
			if len(parts) == 0 {
				parts = append(parts, gmPart{Text: "."})
			}
			add("user", parts...)
		}
	}
	return out
}

// geminiSchema strips JSON Schema keywords the API rejects.
func geminiSchema(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range t {
			switch k {
			case "additionalProperties", "$schema", "default", "examples":
				continue
			}
			out[k] = geminiSchema(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = geminiSchema(t[i])
		}
		return out
	}
	return v
}

// Chat implements Provider.
func (p *Gemini) Chat(ctx context.Context, req Request, onEvent func(Event)) (*Response, error) {
	if p.APIKey == "" {
		return nil, ErrNoAPIKey
	}
	body := map[string]any{"contents": geminiContents(req.Messages)}
	if req.System != "" {
		body["systemInstruction"] = gmContent{Parts: []gmPart{{Text: req.System}}}
	}
	if len(req.Tools) > 0 {
		var decls []map[string]any
		for _, t := range req.Tools {
			d := map[string]any{"name": t.Name, "description": t.Description}
			if props, ok := t.Schema["properties"].(map[string]any); ok && len(props) > 0 {
				d["parameters"] = geminiSchema(t.Schema)
			}
			decls = append(decls, d)
		}
		body["tools"] = []map[string]any{{"functionDeclarations": decls}}
	}
	gen := map[string]any{}
	if req.Temperature != nil {
		gen["temperature"] = *req.Temperature
	}
	if req.MaxTokens > 0 {
		gen["maxOutputTokens"] = req.MaxTokens
	}
	if tc := geminiThinking(req); len(tc) > 0 {
		gen["thinkingConfig"] = tc
	}
	if len(gen) > 0 {
		body["generationConfig"] = gen
	}
	u := p.base() + "/models/" + url.PathEscape(req.Model) + ":streamGenerateContent?alt=sse"
	resp, err := postJSON(ctx, "Gemini", u, map[string]string{"x-goog-api-key": p.APIKey}, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	col := newCollector(onEvent)
	var native []gmPart
	var calls []ToolCall
	err = readSSE(resp.Body, func(_, data string) error {
		var chunk struct {
			Candidates []struct {
				Content      gmContent `json:"content"`
				FinishReason string    `json:"finishReason"`
			} `json:"candidates"`
			UsageMetadata *struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				ThoughtsTokenCount   int `json:"thoughtsTokenCount"`
			} `json:"usageMetadata"`
			PromptFeedback *struct {
				BlockReason string `json:"blockReason"`
			} `json:"promptFeedback"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil
		}
		if chunk.Error != nil {
			return &APIError{Provider: "Gemini", Message: chunk.Error.Message}
		}
		if chunk.PromptFeedback != nil && chunk.PromptFeedback.BlockReason != "" {
			return &APIError{Provider: "Gemini", Message: "the prompt was blocked (" + chunk.PromptFeedback.BlockReason + ")"}
		}
		if u := chunk.UsageMetadata; u != nil {
			col.resp.Usage = Usage{InputTokens: u.PromptTokenCount, OutputTokens: u.CandidatesTokenCount + u.ThoughtsTokenCount}
		}
		for _, c := range chunk.Candidates {
			for _, part := range c.Content.Parts {
				native = append(native, part)
				switch {
				case part.FunctionCall != nil:
					id := part.FunctionCall.ID
					if id == "" {
						id = newCallID()
					}
					calls = append(calls, ToolCall{ID: id, Name: part.FunctionCall.Name, Args: part.FunctionCall.Args})
				case part.Thought:
					col.reasoningDelta(part.Text)
				default:
					col.textDelta(part.Text)
				}
			}
			if c.FinishReason != "" {
				col.resp.StopReason = normalizeFinish(c.FinishReason)
				if c.FinishReason == "SAFETY" || c.FinishReason == "PROHIBITED_CONTENT" {
					col.resp.StopReason = "refusal"
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, c := range calls {
		col.toolCall(c)
	}
	if len(calls) > 0 {
		col.resp.StopReason = "tool_calls"
	}
	col.resp.Native = mergeTextParts(native)
	return col.finish(), nil
}

// mergeTextParts joins consecutive plain text parts so replayed history
// stays compact; parts carrying signatures or calls are kept as is.
func mergeTextParts(parts []gmPart) []gmPart {
	var out []gmPart
	for _, p := range parts {
		plain := p.FunctionCall == nil && p.InlineData == nil && p.ThoughtSignature == ""
		if n := len(out); n > 0 && plain {
			last := &out[n-1]
			if last.FunctionCall == nil && last.InlineData == nil && last.ThoughtSignature == "" && last.Thought == p.Thought {
				last.Text += p.Text
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

// Models implements Provider.
func (p *Gemini) Models(ctx context.Context) ([]ModelInfo, error) {
	if p.APIKey == "" {
		return nil, ErrNoAPIKey
	}
	var out struct {
		Models []struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			InputTokenLimit            int      `json:"inputTokenLimit"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
			Thinking                   bool     `json:"thinking"`
		} `json:"models"`
	}
	if err := getJSON(ctx, "Gemini", p.base()+"/models?pageSize=200", map[string]string{"x-goog-api-key": p.APIKey}, &out); err != nil {
		return nil, err
	}
	var models []ModelInfo
	for _, m := range out.Models {
		ok := false
		for _, g := range m.SupportedGenerationMethods {
			if g == "generateContent" {
				ok = true
			}
		}
		if !ok || strings.Contains(m.Name, "embedding") {
			continue
		}
		models = append(models, ModelInfo{
			ID:            strings.TrimPrefix(m.Name, "models/"),
			Name:          m.DisplayName,
			ContextWindow: m.InputTokenLimit,
			Vision:        true,
			Tools:         true,
			Reasoning:     m.Thinking,
		})
	}
	return models, nil
}

// geminiThinking builds thinkingConfig: Gemini 3 takes a level, 2.5 a
// token budget (Pro can't go below 128).
func geminiThinking(req Request) map[string]any {
	tc := map[string]any{}
	if req.Reasoning && req.Thinking != ThinkingOff {
		tc["includeThoughts"] = true
	}
	switch {
	case req.Thinking == ThinkingAuto:
	case strings.Contains(req.Model, "gemini-3"):
		// Flash takes minimal to high; Pro only low and high.
		level := string(req.Thinking)
		flash := strings.Contains(req.Model, "flash")
		switch {
		case req.Thinking == ThinkingOff && flash:
			level = "minimal"
		case req.Thinking == ThinkingOff:
			level = "low"
		case req.Thinking == ThinkingMedium && !flash:
			level = "high"
		}
		tc["thinkingLevel"] = level
	default:
		budget := thinkingBudget(req.Thinking)
		if req.Thinking == ThinkingOff && strings.Contains(req.Model, "pro") {
			budget = 128
		}
		tc["thinkingBudget"] = budget
	}
	return tc
}
