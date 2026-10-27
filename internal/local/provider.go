package local

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/mehran9675/pocket-agent/internal/llm"
)

// Provider exposes the engine as an llm.Provider; model IDs are GGUF
// file names inside the models directory.
type Provider struct {
	Engine Engine
	Store  *Store
	// Options returns the current load options and sampling settings.
	Options func() (LoadOptions, GenParams)
	// OnLoad, when set, is told when a model starts and finishes loading.
	OnLoad func(model string, loading bool)

	mu sync.Mutex
}

// Chat implements llm.Provider. Tools are offered through a prompt
// protocol (<tool_call>{json}</tool_call>, the Hermes/Qwen convention)
// since llama.cpp's built-in templates have no tool support.
func (p *Provider) Chat(ctx context.Context, req llm.Request, onEvent func(llm.Event)) (*llm.Response, error) {
	if !p.Engine.Available() {
		return nil, ErrUnavailable
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	path := filepath.Join(p.Store.Dir(), req.Model)
	opts, gen := LoadOptions{GPULayers: -1}, DefaultGenParams()
	if p.Options != nil {
		opts, gen = p.Options()
	}
	if p.Engine.Loaded() != path {
		if p.OnLoad != nil {
			p.OnLoad(req.Model, true)
		}
		err := p.Engine.Load(path, opts)
		if p.OnLoad != nil {
			p.OnLoad(req.Model, false)
		}
		if err != nil {
			return nil, err
		}
	}
	if req.Temperature != nil {
		gen.Temperature = float32(*req.Temperature)
	}
	if req.MaxTokens > 0 {
		gen.MaxTokens = req.MaxTokens
	}

	msgs := templateMessages(req)
	prompt, err := p.prompt(msgs, req.Thinking)
	if err != nil {
		return nil, err
	}
	// Drop the oldest turns until the prompt leaves room to answer.
	budget := p.Engine.ContextSize() - min(gen.MaxTokens, p.Engine.ContextSize()/3)
	for len(msgs) > 2 && p.Engine.CountTokens(prompt) > budget {
		msgs = append(msgs[:1], msgs[2:]...)
		if prompt, err = p.prompt(msgs, req.Thinking); err != nil {
			return nil, err
		}
	}

	col := &collector{onEvent: onEvent, tools: len(req.Tools) > 0}
	stats, err := p.Engine.Generate(ctx, prompt, gen, func(piece string) bool {
		col.feed(piece)
		return true
	})
	if err != nil && ctx.Err() == nil {
		return nil, err
	}
	resp := col.finish()
	resp.Usage = llm.Usage{InputTokens: stats.PromptTokens, OutputTokens: stats.GeneratedTokens}
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// prompt formats msgs. With thinking off it uses the convention of
// hybrid models (Qwen3, SmolLM3, ...): /no_think on the last user turn
// and an empty thinking block to start the reply.
func (p *Provider) prompt(msgs []TemplateMessage, thinking llm.Thinking) (string, error) {
	if thinking != llm.ThinkingOff {
		return p.Engine.ApplyTemplate(msgs, true)
	}
	msgs = append([]TemplateMessage(nil), msgs...)
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			msgs[i].Content += " /no_think"
			break
		}
	}
	prompt, err := p.Engine.ApplyTemplate(msgs, true)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(strings.TrimSpace(prompt), "</think>") {
		prompt += "<think>\n\n</think>\n\n"
	}
	return prompt, nil
}

// Models implements llm.Provider: the downloaded models.
func (p *Provider) Models(context.Context) ([]llm.ModelInfo, error) {
	var out []llm.ModelInfo
	for _, m := range p.Store.Installed() {
		out = append(out, llm.ModelInfo{ID: m.File, Name: m.Name, Tools: true, Reasoning: m.Reasoning, ContextWindow: m.Context})
	}
	return out, nil
}

func toolPrompt(tools []llm.ToolSpec) string {
	var b strings.Builder
	b.WriteString("\n\n# Tools\n\nYou may call these tools when they help. To call one, reply with a single block:\n")
	b.WriteString("<tool_call>\n{\"name\": \"tool_name\", \"arguments\": {...}}\n</tool_call>\n")
	b.WriteString("and stop; the result arrives in the next message. Write a normal answer when no tool is needed.\n\nAvailable tools:\n")
	for _, t := range tools {
		schema, _ := json.Marshal(t.Schema)
		fmt.Fprintf(&b, "- %s: %s\n  arguments schema: %s\n", t.Name, t.Description, schema)
	}
	return b.String()
}

func templateMessages(req llm.Request) []TemplateMessage {
	system := req.System
	if len(req.Tools) > 0 {
		system += toolPrompt(req.Tools)
	}
	var msgs []TemplateMessage
	if system != "" {
		msgs = append(msgs, TemplateMessage{Role: "system", Content: system})
	}
	for _, m := range req.Messages {
		switch m.Role {
		case llm.RoleTool:
			// Small templates know no tool role; results go back as user
			// turns, tagged.
			content := fmt.Sprintf("<tool_response name=%q>\n%s\n</tool_response>", m.ToolName, m.Text())
			if n := len(msgs); n > 0 && msgs[n-1].Role == "user" && strings.HasPrefix(msgs[n-1].Content, "<tool_response") {
				msgs[n-1].Content += "\n" + content
				continue
			}
			msgs = append(msgs, TemplateMessage{Role: "user", Content: content})
		case llm.RoleAssistant:
			content := m.Text()
			for _, tc := range m.ToolCalls {
				content += fmt.Sprintf("\n<tool_call>\n{\"name\": %q, \"arguments\": %s}\n</tool_call>", tc.Name, string(tc.Args))
			}
			msgs = append(msgs, TemplateMessage{Role: "assistant", Content: strings.TrimSpace(content)})
		default:
			text := m.Text()
			for _, part := range m.Parts {
				if part.Kind == llm.PartImage {
					text += "\n[An image was attached, but this on-device model can't see images.]"
				}
			}
			msgs = append(msgs, TemplateMessage{Role: string(m.Role), Content: text})
		}
	}
	return msgs
}

var toolCallRE = regexp.MustCompile(`(?s)<tool_call>\s*(\{.*?\})\s*</tool_call>`)

// collector splits thinking and tool calls out of the raw text stream.
type collector struct {
	onEvent func(llm.Event)
	tools   bool
	think   llm.ThinkSplitter
	text    strings.Builder
	reason  strings.Builder
	// held is reply text not yet shown because it may start a tool call.
	held string
}

func (c *collector) emit(kind llm.EventKind, s string) {
	if s != "" && c.onEvent != nil {
		c.onEvent(llm.Event{Kind: kind, Text: s})
	}
}

func (c *collector) feed(piece string) {
	t, r := c.think.Feed(piece)
	c.reason.WriteString(r)
	c.emit(llm.EventReasoning, r)
	c.reply(t)
}

func (c *collector) reply(t string) {
	c.text.WriteString(t)
	if !c.tools {
		c.emit(llm.EventText, t)
		return
	}
	// Show text up to a possible "<tool_call>" start; hold the rest.
	s := c.held + t
	if i := strings.Index(s, "<tool_call>"); i >= 0 {
		c.emit(llm.EventText, s[:i])
		c.held = s[i:]
		return
	}
	keep := 0
	for n := min(len("<tool_call>")-1, len(s)); n > 0; n-- {
		if strings.HasSuffix(s, "<tool_call>"[:n]) {
			keep = n
			break
		}
	}
	c.emit(llm.EventText, s[:len(s)-keep])
	c.held = s[len(s)-keep:]
}

func (c *collector) finish() *llm.Response {
	t, r := c.think.Flush()
	c.reason.WriteString(r)
	c.emit(llm.EventReasoning, r)
	c.reply(t)
	full := c.text.String()
	resp := &llm.Response{Reasoning: c.reason.String(), StopReason: "stop"}
	if c.tools {
		for i, m := range toolCallRE.FindAllStringSubmatch(full, -1) {
			var call struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			if json.Unmarshal([]byte(m[1]), &call) != nil || call.Name == "" {
				continue
			}
			tc := llm.ToolCall{ID: fmt.Sprintf("local_%d", i), Name: call.Name, Args: call.Arguments}
			if len(tc.Args) == 0 {
				tc.Args = json.RawMessage("{}")
			}
			resp.ToolCalls = append(resp.ToolCalls, tc)
			if c.onEvent != nil {
				c.onEvent(llm.Event{Kind: llm.EventToolCall, ToolCall: tc})
			}
		}
		if len(resp.ToolCalls) > 0 {
			resp.StopReason = "tool_calls"
		} else if c.held != "" {
			// An unfinished tag: show it as text after all.
			c.emit(llm.EventText, c.held)
		}
		full = strings.TrimSpace(toolCallRE.ReplaceAllString(full, ""))
		if i := strings.Index(full, "<tool_call>"); i >= 0 && len(resp.ToolCalls) > 0 {
			full = strings.TrimSpace(full[:i])
		}
	}
	resp.Text = full
	return resp
}
