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

// looping calls the calculator forever, using 1000 tokens a call.
type looping struct{ calls int }

func (*looping) Models(context.Context) ([]llm.ModelInfo, error) { return nil, nil }
func (l *looping) Chat(context.Context, llm.Request, func(llm.Event)) (*llm.Response, error) {
	l.calls++
	return &llm.Response{
		ToolCalls:  []llm.ToolCall{{ID: fmt.Sprint("c", l.calls), Name: "calculator", Args: json.RawMessage(`{"expression":"1+1"}`)}},
		StopReason: "tool_calls", Usage: llm.Usage{InputTokens: 900, OutputTokens: 100},
	}, nil
}

func TestRunStopsAtTokenBudget(t *testing.T) {
	calc, _ := tools.Builtin("calculator")
	p := &looping{}
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
