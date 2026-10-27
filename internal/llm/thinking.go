package llm

import "strings"

// Thinking is how much a reasoning model thinks before it answers.
type Thinking string

const (
	// ThinkingAuto leaves it to the model.
	ThinkingAuto   Thinking = ""
	ThinkingOff    Thinking = "off"
	ThinkingLow    Thinking = "low"
	ThinkingMedium Thinking = "medium"
	ThinkingHigh   Thinking = "high"
)

// ThinkingLevels lists the levels in display order.
var ThinkingLevels = []Thinking{ThinkingAuto, ThinkingOff, ThinkingLow, ThinkingMedium, ThinkingHigh}

// Label names the level for display.
func (t Thinking) Label() string {
	switch t {
	case ThinkingOff:
		return "Off"
	case ThinkingLow:
		return "Low"
	case ThinkingMedium:
		return "Medium"
	case ThinkingHigh:
		return "High"
	}
	return "Auto"
}

// ThinkingStyle is how an OpenAI-compatible server takes a thinking
// level; servers differ, and some reject fields they don't know.
type ThinkingStyle string

const (
	// ThinkingNone: the server has no switch; levels aren't offered.
	ThinkingNone ThinkingStyle = ""
	// ThinkingOpenAI: reasoning_effort, with "minimal" for off.
	ThinkingOpenAI ThinkingStyle = "openai"
	// ThinkingEffort: reasoning_effort, with "none" for off (Ollama,
	// DeepInfra, Fireworks, Groq, ...).
	ThinkingEffort ThinkingStyle = "effort"
	// ThinkingOpenRouter: the reasoning object.
	ThinkingOpenRouter ThinkingStyle = "openrouter"
	// ThinkingTemplate: chat_template_kwargs.enable_thinking plus
	// reasoning_effort (vLLM, SGLang, llama.cpp server).
	ThinkingTemplate ThinkingStyle = "template"
	// ThinkingQwen: DashScope's enable_thinking.
	ThinkingQwen ThinkingStyle = "qwen"
	// ThinkingZAI: Z.ai's thinking object.
	ThinkingZAI ThinkingStyle = "zai"
)

// apply adds the fields for t to an OpenAI-compatible request body.
func (s ThinkingStyle) apply(body map[string]any, t Thinking) {
	if t == ThinkingAuto {
		return
	}
	effort := string(t)
	switch s {
	case ThinkingOpenAI:
		if t == ThinkingOff {
			effort = "minimal"
		}
		body["reasoning_effort"] = effort
	case ThinkingEffort:
		if t == ThinkingOff {
			effort = "none"
		}
		body["reasoning_effort"] = effort
	case ThinkingOpenRouter:
		if t == ThinkingOff {
			body["reasoning"] = map[string]any{"enabled": false}
		} else {
			body["reasoning"] = map[string]any{"effort": effort}
		}
	case ThinkingTemplate:
		body["chat_template_kwargs"] = map[string]any{"enable_thinking": t != ThinkingOff}
		if t != ThinkingOff {
			body["reasoning_effort"] = effort
		}
	case ThinkingQwen:
		body["enable_thinking"] = t != ThinkingOff
		if budget := thinkingBudget(t); budget > 0 {
			body["thinking_budget"] = budget
		}
	case ThinkingZAI:
		kind := "enabled"
		if t == ThinkingOff {
			kind = "disabled"
		}
		body["thinking"] = map[string]any{"type": kind}
	}
}

// thinkingBudget is a token budget for APIs that take one.
func thinkingBudget(t Thinking) int {
	switch t {
	case ThinkingLow:
		return 1024
	case ThinkingMedium:
		return 8192
	case ThinkingHigh:
		return 24576
	}
	return 0
}

// CanControlThinking reports whether a provider of this kind and
// preset honors thinking levels.
func CanControlThinking(kind Kind, pre Preset) bool {
	switch kind {
	case KindAnthropic, KindGemini, KindLocal, kindMock:
		return true
	}
	return pre.Thinking != ThinkingNone
}

// LooksLikeReasoning guesses from a model ID whether it thinks before
// answering, for servers whose model lists don't say.
func LooksLikeReasoning(id string) bool {
	id = strings.ToLower(id)
	if i := strings.LastIndex(id, "/"); i >= 0 {
		id = id[i+1:]
	}
	for _, p := range []string{"o1", "o3", "o4", "gpt-5"} {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	for _, s := range []string{
		"reason", "think", "-r1", "r1-", "qwq", "qwen3", "gpt-oss", "magistral",
		"grok-3-mini", "grok-4", "deepseek-v3.1", "deepseek-v3.2", "glm-4.5", "glm-4.6", "glm-5",
		"kimi-k2", "minimax-m", "phi-4-reasoning", "nemotron", "sonar-reasoning", "gemini-2.5", "gemini-3",
	} {
		if strings.Contains(id, s) {
			return true
		}
	}
	return false
}
