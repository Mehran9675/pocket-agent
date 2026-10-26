package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mehran9675/pocket-agent/internal/llm"
)

func TestEvaluate(t *testing.T) {
	cases := map[string]string{
		"1+2*3":            "7",
		"(1+2)*3":          "9",
		"2^3^2":            "512",
		"-2^2":             "-4",
		"sqrt(16)+abs(-2)": "6",
		"5!":               "120",
		"max(1, 7, 3)":     "7",
		"10 % 4":           "2",
		"1e3/4":            "250",
	}
	for in, want := range cases {
		v, err := Evaluate(in)
		if err != nil || FormatNumber(v) != want {
			t.Errorf("%s = %v (%v), want %s", in, FormatNumber(v), err, want)
		}
	}
	if _, err := Evaluate("2 +"); err == nil {
		t.Error("expected error")
	}
}

func TestJavaScriptAndWorkspace(t *testing.T) {
	env := &Env{Workspace: t.TempDir()}
	set := NewSet(Builtins()...)
	res := set.Run(context.Background(), env, llm.ToolCall{Name: "run_javascript", Args: json.RawMessage(`{"code":"writeFile('a/b.txt','hi'); console.log([1,2].map(x=>x*2)); 6*7"}`)})
	if res.IsError || !strings.Contains(res.Text, "[\n  2,\n  4\n]") || !strings.Contains(res.Text, "=> 42") {
		t.Fatalf("js result: %+v", res)
	}
	res = set.Run(context.Background(), env, llm.ToolCall{Name: "read_file", Args: json.RawMessage(`{"path":"../../a/b.txt"}`)})
	if res.Text != "hi" {
		t.Fatalf("read_file escaped or failed: %+v", res)
	}
	res = set.Run(context.Background(), env, llm.ToolCall{Name: "write_file", Args: json.RawMessage(`{"path":"x.md"}`)})
	if !res.IsError || !strings.Contains(res.Text, "content") {
		t.Fatalf("missing arg not reported: %+v", res)
	}
	res = set.Run(context.Background(), env, llm.ToolCall{Name: "run_javascript", Args: json.RawMessage(`{"code":"while(true){}"`)})
	if !res.IsError || !strings.Contains(res.Text, "INVALID_JSON") {
		t.Fatalf("invalid json not reported: %+v", res)
	}
}
