package agent

import (
	"context"
	"encoding/json"
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
