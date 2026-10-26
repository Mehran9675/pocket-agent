package local

import (
	"strings"
	"testing"

	"github.com/mehran9675/pocket-agent/internal/llm"
)

func TestCollectorToolCall(t *testing.T) {
	var shown strings.Builder
	c := &collector{tools: true, onEvent: func(e llm.Event) {
		if e.Kind == llm.EventText {
			shown.WriteString(e.Text)
		}
	}}
	for _, p := range []string{"<think>plan</think>Let me check.\n<tool", "_call>\n{\"name\": \"web_search\", \"arguments\": {\"query\": \"x\"}}\n</tool_call>"} {
		c.feed(p)
	}
	resp := c.finish()
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "web_search" {
		t.Fatalf("calls %+v", resp.ToolCalls)
	}
	if shown.String() != "Let me check.\n" || resp.Text != "Let me check." || resp.Reasoning != "plan" {
		t.Fatalf("shown=%q text=%q reasoning=%q", shown.String(), resp.Text, resp.Reasoning)
	}
}

func TestSplitValidUTF8(t *testing.T) {
	b := []byte("é")
	s, rest := splitValidUTF8(append([]byte("a"), b[0]))
	if s != "a" || len(rest) != 1 {
		t.Fatalf("%q %v", s, rest)
	}
}
