package llm

import (
	"crypto/rand"
	"encoding/hex"
)

// Kind selects the wire protocol of a provider.
type Kind string

const (
	KindOpenAI    Kind = "openai"
	KindAnthropic Kind = "anthropic"
	KindGemini    Kind = "gemini"
	// KindLocal is the on-device engine (package local).
	KindLocal Kind = "local"
	kindMock  Kind = "mock"
)

// Preset is a known provider with its defaults.
type Preset struct {
	ID      string
	Name    string
	Kind    Kind
	BaseURL string
	// KeyOptional is set for servers that need no key (local ones).
	KeyOptional bool
	// KeyHint tells the user where keys come from.
	KeyHint string
	// LAN marks servers that usually run on the user's own network.
	LAN bool
	// Models are suggested when the provider can't list its own.
	Models []ModelInfo
	// Headers are sent with every request.
	Headers map[string]string
	// NoStreamUsage disables stream_options for servers that reject it.
	NoStreamUsage bool
}

// Presets lists the providers Pocket Agent knows, in display order.
var Presets = []Preset{
	{
		ID: "anthropic", Name: "Anthropic", Kind: KindAnthropic,
		KeyHint: "console.anthropic.com → API keys",
		Models: []ModelInfo{
			{ID: "claude-opus-5-5", Name: "Claude Opus 5.5", Vision: true, Tools: true, Reasoning: true, ContextWindow: 1_000_000},
			{ID: "claude-sonnet-5-5", Name: "Claude Sonnet 5.5", Vision: true, Tools: true, Reasoning: true, ContextWindow: 1_000_000},
			{ID: "claude-haiku-5-5", Name: "Claude Haiku 5.5", Vision: true, Tools: true, Reasoning: true, ContextWindow: 1_000_000},
			{ID: "claude-fable-5-1", Name: "Claude Fable 5.1", Vision: true, Tools: true, Reasoning: true, ContextWindow: 1_000_000},
		},
	},
	{
		ID: "openai", Name: "OpenAI", Kind: KindOpenAI, BaseURL: "https://api.openai.com/v1",
		KeyHint: "platform.openai.com → API keys",
	},
	{
		ID: "gemini", Name: "Google Gemini", Kind: KindGemini,
		KeyHint: "aistudio.google.com → Get API key",
	},
	{
		ID: "openrouter", Name: "OpenRouter", Kind: KindOpenAI, BaseURL: "https://openrouter.ai/api/v1",
		KeyHint: "openrouter.ai → Keys",
		Headers: map[string]string{"HTTP-Referer": "https://github.com/mehran9675/pocket-agent", "X-Title": "Pocket Agent"},
	},
	{ID: "groq", Name: "Groq", Kind: KindOpenAI, BaseURL: "https://api.groq.com/openai/v1", KeyHint: "console.groq.com → API Keys"},
	{ID: "mistral", Name: "Mistral", Kind: KindOpenAI, BaseURL: "https://api.mistral.ai/v1", KeyHint: "console.mistral.ai → API Keys", NoStreamUsage: true},
	{ID: "deepseek", Name: "DeepSeek", Kind: KindOpenAI, BaseURL: "https://api.deepseek.com/v1", KeyHint: "platform.deepseek.com → API keys"},
	{ID: "xai", Name: "xAI", Kind: KindOpenAI, BaseURL: "https://api.x.ai/v1", KeyHint: "console.x.ai → API Keys"},
	{ID: "together", Name: "Together AI", Kind: KindOpenAI, BaseURL: "https://api.together.xyz/v1", KeyHint: "api.together.ai → Settings → API Keys"},
	{
		ID: "ollama", Name: "Ollama", Kind: KindOpenAI, BaseURL: "http://192.168.1.10:11434/v1",
		KeyOptional: true, LAN: true, KeyHint: "Run `OLLAMA_HOST=0.0.0.0 ollama serve` on a computer on your network",
	},
	{
		ID: "lmstudio", Name: "LM Studio", Kind: KindOpenAI, BaseURL: "http://192.168.1.10:1234/v1",
		KeyOptional: true, LAN: true, KeyHint: "Developer tab → Start server, and enable “Serve on local network”",
	},
	{
		ID: "custom", Name: "Custom (OpenAI-compatible)", Kind: KindOpenAI, BaseURL: "https://",
		KeyOptional: true, KeyHint: "Any server speaking the Chat Completions API: vLLM, llama.cpp server, LiteLLM, ...",
	},
}

// PresetByID returns the preset with id.
func PresetByID(id string) (Preset, bool) {
	for _, p := range Presets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}

// mockFactory is set by the devmock build to provide an offline demo
// provider for UI development.
var mockFactory func() Provider

// New builds a provider for a preset kind.
func New(kind Kind, name, baseURL, apiKey string, extra Preset) Provider {
	if kind == kindMock && mockFactory != nil {
		return mockFactory()
	}
	switch kind {
	case KindAnthropic:
		return &Anthropic{APIKey: apiKey, BaseURL: baseURL}
	case KindGemini:
		return &Gemini{APIKey: apiKey, BaseURL: baseURL}
	default:
		return &OpenAI{
			Name:          name,
			BaseURL:       baseURL,
			APIKey:        apiKey,
			Headers:       extra.Headers,
			KeyOptional:   extra.KeyOptional,
			Official:      extra.ID == "openai",
			NoStreamUsage: extra.NoStreamUsage,
		}
	}
}

func newCallID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "call_" + hex.EncodeToString(b[:])
}
