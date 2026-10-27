package llm

import (
	"encoding/json"
	"testing"
)

func TestThinkingStyles(t *testing.T) {
	cases := []struct {
		style ThinkingStyle
		level Thinking
		want  string
	}{
		{ThinkingOpenAI, ThinkingOff, `{"reasoning_effort":"minimal"}`},
		{ThinkingOpenAI, ThinkingHigh, `{"reasoning_effort":"high"}`},
		{ThinkingEffort, ThinkingOff, `{"reasoning_effort":"none"}`},
		{ThinkingOpenRouter, ThinkingOff, `{"reasoning":{"enabled":false}}`},
		{ThinkingOpenRouter, ThinkingLow, `{"reasoning":{"effort":"low"}}`},
		{ThinkingTemplate, ThinkingOff, `{"chat_template_kwargs":{"enable_thinking":false}}`},
		{ThinkingQwen, ThinkingMedium, `{"enable_thinking":true,"thinking_budget":8192}`},
		{ThinkingZAI, ThinkingOff, `{"thinking":{"type":"disabled"}}`},
		{ThinkingNone, ThinkingOff, `{}`},
		{ThinkingOpenAI, ThinkingAuto, `{}`},
	}
	for _, c := range cases {
		body := map[string]any{}
		c.style.apply(body, c.level)
		got, _ := json.Marshal(body)
		if string(got) != c.want {
			t.Errorf("%s/%s: got %s, want %s", c.style, c.level, got, c.want)
		}
	}
}

func TestGeminiThinking(t *testing.T) {
	if tc := geminiThinking(Request{Model: "gemini-2.5-flash", Thinking: ThinkingOff}); tc["thinkingBudget"] != 0 {
		t.Errorf("2.5 flash off: %v", tc)
	}
	if tc := geminiThinking(Request{Model: "gemini-2.5-pro", Thinking: ThinkingOff}); tc["thinkingBudget"] != 128 {
		t.Errorf("2.5 pro off: %v", tc)
	}
	if tc := geminiThinking(Request{Model: "gemini-3-pro-preview", Thinking: ThinkingMedium}); tc["thinkingLevel"] != "high" {
		t.Errorf("3 pro medium: %v", tc)
	}
	if tc := geminiThinking(Request{Model: "gemini-3-flash", Thinking: ThinkingOff, Reasoning: true}); tc["thinkingLevel"] != "minimal" || tc["includeThoughts"] != nil {
		t.Errorf("3 flash off: %v", tc)
	}
}

func TestLooksLikeReasoning(t *testing.T) {
	for _, id := range []string{"o3-mini", "gpt-5", "deepseek/deepseek-r1", "Qwen/Qwen3-32B", "openai/gpt-oss-120b", "moonshotai/Kimi-K2-Thinking"} {
		if !LooksLikeReasoning(id) {
			t.Errorf("%s not detected", id)
		}
	}
	for _, id := range []string{"gpt-4o", "meta-llama/Llama-3.3-70B-Instruct", "mistral-large-latest"} {
		if LooksLikeReasoning(id) {
			t.Errorf("%s detected", id)
		}
	}
}
