package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sse(lines ...string) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString("data: " + l + "\n\n")
	}
	return b.String()
}

func TestOpenAIStreamWithToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("missing auth header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sse(
			`{"choices":[{"delta":{"content":"<thi"}}]}`,
			`{"choices":[{"delta":{"content":"nk>pondering</think>\n\nHel"}}]}`,
			`{"choices":[{"delta":{"content":"lo"}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"calc","arguments":"{\"x\":"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"1}"}}]},"finish_reason":"tool_calls"}]}`,
			`{"choices":[],"usage":{"prompt_tokens":7,"completion_tokens":3}}`,
			`[DONE]`,
		)))
	}))
	defer srv.Close()
	p := &OpenAI{Name: "test", BaseURL: srv.URL, APIKey: "k"}
	var text, reason strings.Builder
	resp, err := p.Chat(context.Background(), Request{Model: "m", Messages: []Message{{Role: RoleUser, Parts: []Part{TextPart("hi")}}}}, func(e Event) {
		switch e.Kind {
		case EventText:
			text.WriteString(e.Text)
		case EventReasoning:
			reason.WriteString(e.Text)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if text.String() != "Hello" || resp.Text != "Hello" {
		t.Errorf("text = %q / %q", text.String(), resp.Text)
	}
	if reason.String() != "pondering" {
		t.Errorf("reasoning = %q", reason.String())
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "calc" || string(resp.ToolCalls[0].Args) != `{"x":1}` {
		t.Errorf("tool calls = %+v", resp.ToolCalls)
	}
	if resp.Usage.InputTokens != 7 || resp.StopReason != "tool_calls" {
		t.Errorf("usage/stop = %+v %s", resp.Usage, resp.StopReason)
	}
}

func TestOpenAIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer srv.Close()
	p := &OpenAI{Name: "X", BaseURL: srv.URL, APIKey: "k"}
	_, err := p.Chat(context.Background(), Request{Model: "m"}, nil)
	if err == nil || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("err = %v", err)
	}
}

func TestGeminiStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "gemini-x:streamGenerateContent") {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Write([]byte(sse(
			`{"candidates":[{"content":{"role":"model","parts":[{"text":"hmm","thought":true}]}}]}`,
			`{"candidates":[{"content":{"role":"model","parts":[{"text":"Hi "},{"text":"there"}]}}]}`,
			`{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"f","args":{"a":1}},"thoughtSignature":"sig"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":2}}`,
		)))
	}))
	defer srv.Close()
	p := &Gemini{APIKey: "k", BaseURL: srv.URL}
	resp, err := p.Chat(context.Background(), Request{Model: "gemini-x", Messages: []Message{{Role: RoleUser, Parts: []Part{TextPart("q")}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "Hi there" || resp.Reasoning != "hmm" || len(resp.ToolCalls) != 1 || resp.StopReason != "tool_calls" {
		t.Errorf("resp = %+v", resp)
	}
	native := resp.Native.([]gmPart)
	if len(native) != 3 || native[2].ThoughtSignature != "sig" {
		t.Errorf("native parts = %+v", native)
	}
}

func TestThinkSplitterChunked(t *testing.T) {
	var s ThinkSplitter
	in := "<think>a<b</thi" + "nk>reply <x>"
	var text, reason string
	for _, ch := range in {
		t, r := s.Feed(string(ch))
		text += t
		reason += r
	}
	t2, r2 := s.Flush()
	text, reason = text+t2, reason+r2
	if text != "reply <x>" || reason != "a<b" {
		t.Errorf("text=%q reason=%q", text, reason)
	}
}
