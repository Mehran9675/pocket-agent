// Package agent runs the model ↔ tool loop shared by chat, agent mode
// and deep research: stream a reply, run the tools it asks for (with
// the user's approval where needed), feed the results back, repeat.
package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

// Observer receives a run's progress. Methods are called on the run's
// goroutine, in order.
type Observer interface {
	// StepStart is called before each model call.
	StepStart(step int)
	Text(delta string)
	Reasoning(delta string)
	// ToolCall announces a call, before approval.
	ToolCall(call llm.ToolCall, tool *tools.Tool)
	// ToolStatus reports approval and execution progress.
	ToolRunning(call llm.ToolCall)
	ToolDenied(call llm.ToolCall)
	ToolResult(call llm.ToolCall, res tools.Result, d time.Duration)
	Usage(u llm.Usage)
}

// Run is one agentic turn.
type Run struct {
	Provider llm.Provider
	Model    string
	System   string
	Messages []llm.Message
	Tools    *tools.Set
	Env      *tools.Env
	MaxSteps int
	// TokenBudget stops the run, between steps, once its model calls
	// have used this many tokens; 0 is no limit.
	TokenBudget int
	Reasoning   bool
	// Thinking sets how much reasoning models think.
	Thinking  llm.Thinking
	MaxTokens int
	// Approve decides whether a tool needing approval may run; it may
	// block (waiting for the user). Nil approves everything.
	Approve func(ctx context.Context, call llm.ToolCall, tool *tools.Tool) bool
	// Final asks for the answer in one step: tool calls in the reply
	// are dropped instead of run.
	Final bool
	// NeedsApproval overrides the tool's own Approval flag.
	NeedsApproval func(tool *tools.Tool) bool
	Observer      Observer
}

// ErrMaxSteps is returned when the run used up its steps.
var ErrMaxSteps = errors.New("stopped after reaching the step limit")

// ErrTokenBudget is returned when the run used up its token budget.
var ErrTokenBudget = errors.New("stopped after using its token budget")

// ErrRefused is returned when the model declined the request.
var ErrRefused = errors.New("the model declined to answer this request")

// Execute runs until the model answers without calling tools, the step
// limit is hit, or ctx ends. It returns the messages appended to the
// conversation.
func (r *Run) Execute(ctx context.Context) ([]llm.Message, error) {
	maxSteps := r.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 12
	}
	if r.Final {
		maxSteps = 1
	}
	msgs := append([]llm.Message(nil), r.Messages...)
	start := len(msgs)
	var specs []llm.ToolSpec
	if r.Tools != nil {
		specs = r.Tools.Specs()
	}
	used := 0
	for step := 0; step < maxSteps; step++ {
		// Checked between steps, when every tool call has its result,
		// so the conversation can be continued later.
		if r.TokenBudget > 0 && used >= r.TokenBudget {
			return msgs[start:], ErrTokenBudget
		}
		r.Observer.StepStart(step)
		req := llm.Request{
			Model:     r.Model,
			System:    r.System,
			Messages:  CompactToolResults(msgs, keepToolResults),
			Tools:     specs,
			Reasoning: r.Reasoning,
			Thinking:  r.Thinking,
			MaxTokens: r.MaxTokens,
		}
		resp, err := chat(ctx, r.Provider, req, func(ev llm.Event) {
			switch ev.Kind {
			case llm.EventText:
				r.Observer.Text(ev.Text)
			case llm.EventReasoning:
				r.Observer.Reasoning(ev.Text)
			}
		})
		if err != nil {
			return msgs[start:], err
		}
		r.Observer.Usage(resp.Usage)
		used += spent(req, resp)
		assistant := llm.Message{Role: llm.RoleAssistant, ToolCalls: resp.ToolCalls, Native: resp.Native}
		if resp.Text != "" {
			assistant.Parts = []llm.Part{llm.TextPart(resp.Text)}
		}
		msgs = append(msgs, assistant)
		switch resp.StopReason {
		case "refusal":
			return msgs[start:], ErrRefused
		case "length":
			if len(resp.ToolCalls) > 0 {
				// Tool input cut off by the output limit can't be trusted.
				return msgs[start:], errors.New("the reply hit the output limit in the middle of a tool call")
			}
		}
		if r.Final && len(resp.ToolCalls) > 0 {
			assistant.ToolCalls, assistant.Native = nil, nil
			msgs[len(msgs)-1] = assistant
			if resp.Text == "" {
				return msgs[start:], errors.New("the model tried to keep working instead of writing up its results")
			}
		}
		if len(assistant.ToolCalls) == 0 {
			return msgs[start:], nil
		}
		for _, call := range resp.ToolCalls {
			res, ran := r.runTool(ctx, call)
			if ctx.Err() != nil {
				return msgs[start:], ctx.Err()
			}
			text := res.Text
			if !ran {
				text = "The user declined this tool call. Don't retry it; continue without it or ask the user."
			}
			msgs = append(msgs, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Parts:      []llm.Part{llm.TextPart(text)},
				IsError:    res.IsError || !ran,
			})
		}
	}
	return msgs[start:], ErrMaxSteps
}

func (r *Run) runTool(ctx context.Context, call llm.ToolCall) (tools.Result, bool) {
	tool, ok := r.Tools.Get(call.Name)
	r.Observer.ToolCall(call, tool)
	if !ok {
		res := tools.Errorf("unknown tool %q", call.Name)
		r.Observer.ToolResult(call, res, 0)
		return res, true
	}
	needs := tool.Approval
	if r.NeedsApproval != nil {
		needs = r.NeedsApproval(tool)
	}
	if needs && r.Approve != nil && !r.Approve(ctx, call, tool) {
		r.Observer.ToolDenied(call)
		return tools.Result{}, false
	}
	r.Observer.ToolRunning(call)
	t0 := time.Now()
	tctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	res := r.Tools.Run(tctx, r.Env, call)
	cancel()
	if errors.Is(tctx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
		res = tools.Errorf("%s timed out", call.Name)
	}
	r.Observer.ToolResult(call, res, time.Since(t0))
	return res, true
}

// Complete runs a single tool-less prompt and returns the text - for
// titles, research planning and summaries.
func Complete(ctx context.Context, p llm.Provider, model, system, prompt string, maxTokens int) (string, error) {
	resp, err := chat(ctx, p, llm.Request{
		Model:     model,
		System:    system,
		Messages:  []llm.Message{{Role: llm.RoleUser, Parts: []llm.Part{llm.TextPart(prompt)}}},
		MaxTokens: maxTokens,
	}, nil)
	if err != nil {
		return "", err
	}
	if resp.StopReason == "refusal" {
		return "", ErrRefused
	}
	return resp.Text, nil
}

// FriendlyError turns errors into a sentence for the UI.
func FriendlyError(err error) string {
	var api *llm.APIError
	switch {
	case err == nil:
		return ""
	case errors.As(err, &api):
		return api.Error()
	case errors.Is(err, llm.ErrNoAPIKey):
		return "Add an API key for this provider in Settings → Models."
	case errors.Is(err, context.DeadlineExceeded):
		return "The request timed out."
	}
	return fmt.Sprint(err)
}

// chat runs a model call, timing its output for Usage.GenMillis.
func chat(ctx context.Context, p llm.Provider, req llm.Request, onEvent func(llm.Event)) (*llm.Response, error) {
	var first time.Time
	resp, err := p.Chat(ctx, req, func(ev llm.Event) {
		if first.IsZero() {
			first = time.Now()
		}
		if onEvent != nil {
			onEvent(ev)
		}
	})
	if resp != nil && !first.IsZero() {
		resp.Usage.GenMillis = time.Since(first).Milliseconds()
	}
	return resp, err
}

// keepToolResults is how many of the latest tool results are sent in
// full; older ones are trimmed.
const keepToolResults = 6

// spent is the tokens a step used: what the provider reported, or an
// estimate of four characters a token when it reported nothing, so a
// budget still holds with servers that leave usage out.
func spent(req llm.Request, resp *llm.Response) int {
	if n := resp.Usage.InputTokens + resp.Usage.OutputTokens; n > 0 {
		return n
	}
	chars := len(req.System) + len(resp.Text)
	for _, m := range req.Messages {
		chars += len(m.Text())
		for _, c := range m.ToolCalls {
			chars += len(c.Args)
		}
	}
	for _, c := range resp.ToolCalls {
		chars += len(c.Args)
	}
	for _, t := range req.Tools {
		chars += 200 + len(t.Description)
	}
	return chars / 4
}

// trimmedResult is how much of an older tool result is kept.
const trimmedResult = 1500

// CompactToolResults returns msgs with all but the last keep tool
// results cut to an excerpt. Every step resends the whole conversation,
// so without this a long run's input grows with each page it reads and
// its total use grows quadratically. The model can run a tool again for
// anything it needs in full.
func CompactToolResults(msgs []llm.Message, keep int) []llm.Message {
	seen := 0
	var out []llm.Message
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != llm.RoleTool {
			continue
		}
		seen++
		if seen <= keep {
			continue
		}
		text := msgs[i].Text()
		if len([]rune(text)) <= trimmedResult {
			continue
		}
		if out == nil {
			out = append([]llm.Message(nil), msgs...)
		}
		m := out[i]
		m.Parts = []llm.Part{llm.TextPart(string([]rune(text)[:trimmedResult]) + fmt.Sprintf("\n\n[%d more characters trimmed from this earlier result to save context; run the tool again if you need them]", len([]rune(text))-trimmedResult))}
		out[i] = m
	}
	if out == nil {
		return msgs
	}
	return out
}
