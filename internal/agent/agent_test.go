package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

// scripted replies with a tool call first, then text.
type scripted struct{ calls int }

func (s *scripted) Models(context.Context) ([]llm.ModelInfo, error) { return nil, nil }

func (s *scripted) Chat(_ context.Context, req llm.Request, on func(llm.Event)) (*llm.Response, error) {
	s.calls++
	if s.calls == 1 {
		return &llm.Response{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "calculator", Args: json.RawMessage(`{"expression":"6*7"}`)}}, StopReason: "tool_calls"}, nil
	}
	last := req.Messages[len(req.Messages)-1]
	if last.Role != llm.RoleTool || last.Text() != "6*7 = 42" {
		return nil, context.DeadlineExceeded
	}
	on(llm.Event{Kind: llm.EventText, Text: "42"})
	return &llm.Response{Text: "42", StopReason: "stop"}, nil
}

type nopObserver struct{ results []tools.Result }

func (*nopObserver) StepStart(int)                      {}
func (*nopObserver) Text(string)                        {}
func (*nopObserver) Reasoning(string)                   {}
func (*nopObserver) ToolCall(llm.ToolCall, *tools.Tool) {}
func (*nopObserver) ToolRunning(llm.ToolCall)           {}
func (*nopObserver) ToolDenied(llm.ToolCall)            {}
func (o *nopObserver) ToolResult(_ llm.ToolCall, r tools.Result, _ time.Duration) {
	o.results = append(o.results, r)
}
func (*nopObserver) Usage(llm.Usage) {}

func TestRunExecutesToolsAndFinishes(t *testing.T) {
	calc, _ := tools.Builtin("calculator")
	obs := &nopObserver{}
	r := &Run{Provider: &scripted{}, Model: "m", Tools: tools.NewSet(calc), Observer: obs,
		Messages: []llm.Message{{Role: llm.RoleUser, Parts: []llm.Part{llm.TextPart("6*7?")}}}}
	out, err := r.Execute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[2].Text() != "42" || len(obs.results) != 1 {
		t.Fatalf("out=%+v results=%+v", out, obs.results)
	}
}

func TestRunDeniedTool(t *testing.T) {
	del, _ := tools.Builtin("delete_file")
	p := &denyProvider{}
	r := &Run{Provider: p, Model: "m", Tools: tools.NewSet(del), Observer: &nopObserver{},
		Approve: func(context.Context, llm.ToolCall, *tools.Tool) bool { return false }}
	if _, err := r.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !p.sawDenial {
		t.Fatal("the model wasn't told about the denial")
	}
}

type denyProvider struct{ sawDenial bool }

func (*denyProvider) Models(context.Context) ([]llm.ModelInfo, error) { return nil, nil }
func (d *denyProvider) Chat(_ context.Context, req llm.Request, _ func(llm.Event)) (*llm.Response, error) {
	if n := len(req.Messages); n > 0 && req.Messages[n-1].Role == llm.RoleTool {
		d.sawDenial = req.Messages[n-1].IsError
		return &llm.Response{Text: "ok"}, nil
	}
	return &llm.Response{ToolCalls: []llm.ToolCall{{ID: "d", Name: "delete_file", Args: json.RawMessage(`{"path":"x"}`)}}}, nil
}

func TestAssistantMessagesSequence(t *testing.T) {
	m := &store.Message{Role: llm.RoleAssistant, Blocks: []store.Block{
		{Kind: store.BlockText, Text: "Let me look."},
		{Kind: store.BlockReasoning, Text: "hidden"},
		{Kind: store.BlockTool, Tool: &store.ToolRun{CallID: "a", Name: "web_search", Args: `{"query":"q"}`, Result: "r1", Status: store.ToolDone}},
		{Kind: store.BlockTool, Tool: &store.ToolRun{CallID: "b", Name: "fetch_url", Args: `{"url":"u"}`, Status: store.ToolDenied}},
		{Kind: store.BlockText, Text: "Answer."},
	}}
	got := assistantMessages(m)
	roles := []llm.Role{}
	for _, x := range got {
		roles = append(roles, x.Role)
	}
	want := []llm.Role{llm.RoleAssistant, llm.RoleTool, llm.RoleTool, llm.RoleAssistant}
	if len(roles) != len(want) {
		t.Fatalf("roles %v", roles)
	}
	for i := range want {
		if roles[i] != want[i] {
			t.Fatalf("roles %v", roles)
		}
	}
	if len(got[0].ToolCalls) != 2 || got[0].Text() != "Let me look." || !got[2].IsError || got[3].Text() != "Answer." {
		t.Fatalf("%+v", got)
	}
}

// budgetProbe calls the calculator forever, using 1000 tokens a call.
type budgetProbe struct{ calls int }

func (*budgetProbe) Models(context.Context) ([]llm.ModelInfo, error) { return nil, nil }
func (l *budgetProbe) Chat(context.Context, llm.Request, func(llm.Event)) (*llm.Response, error) {
	l.calls++
	return &llm.Response{
		ToolCalls:  []llm.ToolCall{{ID: fmt.Sprint("c", l.calls), Name: "calculator", Args: json.RawMessage(`{"expression":"1+1"}`)}},
		StopReason: "tool_calls", Usage: llm.Usage{InputTokens: 900, OutputTokens: 100},
	}, nil
}

func TestRunStopsAtTokenBudget(t *testing.T) {
	calc, _ := tools.Builtin("calculator")
	p := &budgetProbe{}
	r := &Run{Provider: p, Model: "m", Tools: tools.NewSet(calc), Observer: &nopObserver{}, MaxSteps: 50, TokenBudget: 3500,
		Messages: []llm.Message{{Role: llm.RoleUser, Parts: []llm.Part{llm.TextPart("go")}}}}
	out, err := r.Execute(context.Background())
	if !errors.Is(err, ErrTokenBudget) || p.calls != 4 {
		t.Fatalf("err=%v calls=%d", err, p.calls)
	}
	// It stops between steps: the last call's result is in place.
	if last := out[len(out)-1]; last.Role != llm.RoleTool {
		t.Fatalf("ended on %s", last.Role)
	}
}

func TestCompactToolResults(t *testing.T) {
	big := strings.Repeat("x", 5000)
	var msgs []llm.Message
	for i := 0; i < 10; i++ {
		msgs = append(msgs, llm.Message{Role: llm.RoleTool, Parts: []llm.Part{llm.TextPart(big)}})
	}
	out := CompactToolResults(msgs, 6)
	for i, m := range out {
		trimmed := strings.Contains(m.Text(), "trimmed")
		if trimmed != (i < 4) {
			t.Fatalf("message %d trimmed=%v", i, trimmed)
		}
	}
	if msgs[0].Text() != big {
		t.Fatal("input modified")
	}
}

// wrapProbe calls a tool every step and records whether it was asked to wrap up.
type wrapProbe struct{ notes []bool }

func (s *wrapProbe) Models(context.Context) ([]llm.ModelInfo, error) { return nil, nil }

func (s *wrapProbe) Chat(_ context.Context, req llm.Request, _ func(llm.Event)) (*llm.Response, error) {
	last := req.Messages[len(req.Messages)-1]
	s.notes = append(s.notes, strings.Contains(last.Text(), "Write your final answer now"))
	return &llm.Response{ToolCalls: []llm.ToolCall{{ID: fmt.Sprint("c", len(s.notes)), Name: "calculator", Args: json.RawMessage(`{"expression":"1+1"}`)}},
		StopReason: "tool_calls", Usage: llm.Usage{InputTokens: 100}}, nil
}

func TestRunAsksForAnswerBeforeLimits(t *testing.T) {
	calc, _ := tools.Builtin("calculator")
	run := func(steps, budget int) []bool {
		p := &wrapProbe{}
		r := &Run{Provider: p, Model: "m", Tools: tools.NewSet(calc), Observer: &nopObserver{}, MaxSteps: steps, TokenBudget: budget,
			Messages: []llm.Message{{Role: llm.RoleUser, Parts: []llm.Part{llm.TextPart("go")}}}}
		r.Execute(context.Background())
		return p.notes
	}
	if got := fmt.Sprint(run(5, 0)); got != "[false false false true true]" {
		t.Fatalf("steps: %s", got)
	}
	// 1000 tokens at 100 a step: from 750 used it asks.
	if got := fmt.Sprint(run(20, 1000)); got != "[false false false false false false false false true true]" {
		t.Fatalf("budget: %s", got)
	}
}

func TestSystemPromptListsSkillsAndLimits(t *testing.T) {
	p := SystemPrompt(SystemPromptInput{Mode: store.ModeAgent, MaxSteps: 30, TokenBudget: 500000,
		Skills: [][2]string{{"pdf", "Use for PDFs."}}})
	for _, want := range []string{"- pdf: Use for PDFs.", "call use_skill", "at most 30 steps", "about 500k tokens"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt lacks %q:\n%s", want, p)
		}
	}
}
