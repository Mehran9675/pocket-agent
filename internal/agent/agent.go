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
	Provider  llm.Provider
	Model     string
	System    string
	Messages  []llm.Message
	Tools     *tools.Set
	Env       *tools.Env
	MaxSteps  int
	Reasoning bool
	MaxTokens int
	// Approve decides whether a tool needing approval may run; it may
	// block (waiting for the user). Nil approves everything.
	Approve func(ctx context.Context, call llm.ToolCall, tool *tools.Tool) bool
	// NeedsApproval overrides the tool's own Approval flag.
	NeedsApproval func(tool *tools.Tool) bool
	Observer      Observer
}

// ErrMaxSteps is returned when the run used up its steps.
var ErrMaxSteps = errors.New("stopped after reaching the step limit")

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
	msgs := append([]llm.Message(nil), r.Messages...)
	start := len(msgs)
	var specs []llm.ToolSpec
	if r.Tools != nil {
		specs = r.Tools.Specs()
	}
	for step := 0; step < maxSteps; step++ {
		r.Observer.StepStart(step)
		req := llm.Request{
			Model:     r.Model,
			System:    r.System,
			Messages:  msgs,
			Tools:     specs,
			Reasoning: r.Reasoning,
			MaxTokens: r.MaxTokens,
		}
		resp, err := r.Provider.Chat(ctx, req, func(ev llm.Event) {
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
		if len(resp.ToolCalls) == 0 {
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
	resp, err := p.Chat(ctx, llm.Request{
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
