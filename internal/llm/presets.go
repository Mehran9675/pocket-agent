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
	// Thinking is how an OpenAI-compatible server takes thinking levels.
	Thinking ThinkingStyle
	// Group places the preset in the provider list (GroupCloud, ...).
	Group string
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
		KeyHint: "platform.openai.com → API keys", Thinking: ThinkingOpenAI,
	},
	{
		ID: "gemini", Name: "Google Gemini", Kind: KindGemini,
		KeyHint: "aistudio.google.com → Get API key",
	},
	{ID: "xai", Name: "xAI", Kind: KindOpenAI, BaseURL: "https://api.x.ai/v1", KeyHint: "console.x.ai → API Keys"},
	{ID: "mistral", Name: "Mistral", Kind: KindOpenAI, BaseURL: "https://api.mistral.ai/v1", KeyHint: "console.mistral.ai → API Keys", NoStreamUsage: true},
	{ID: "deepseek", Name: "DeepSeek", Kind: KindOpenAI, BaseURL: "https://api.deepseek.com/v1", KeyHint: "platform.deepseek.com → API keys"},
	{ID: "moonshot", Name: "Moonshot AI (Kimi)", Kind: KindOpenAI, BaseURL: "https://api.moonshot.ai/v1", KeyHint: "platform.moonshot.ai → API Keys"},
	{ID: "zai", Name: "Z.ai (GLM)", Kind: KindOpenAI, BaseURL: "https://api.z.ai/api/paas/v4", KeyHint: "z.ai → API Keys", Thinking: ThinkingZAI},
	{
		ID: "qwen", Name: "Alibaba Cloud (Qwen)", Kind: KindOpenAI, BaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
		KeyHint: "Model Studio console → API Keys", Thinking: ThinkingQwen,
	},
	{ID: "minimax", Name: "MiniMax", Kind: KindOpenAI, BaseURL: "https://api.minimax.io/v1", KeyHint: "platform.minimax.io → API Keys"},
	{
		ID: "perplexity", Name: "Perplexity", Kind: KindOpenAI, BaseURL: "https://api.perplexity.ai",
		KeyHint: "perplexity.ai → Settings → API",
		Models: []ModelInfo{
			{ID: "sonar", Name: "Sonar", ContextWindow: 128_000},
			{ID: "sonar-pro", Name: "Sonar Pro", ContextWindow: 200_000},
			{ID: "sonar-reasoning-pro", Name: "Sonar Reasoning Pro", Reasoning: true, ContextWindow: 128_000},
			{ID: "sonar-deep-research", Name: "Sonar Deep Research", Reasoning: true, ContextWindow: 128_000},
		},
	},
	{
		ID: "cohere", Name: "Cohere", Kind: KindOpenAI, BaseURL: "https://api.cohere.ai/compatibility/v1",
		KeyHint: "dashboard.cohere.com → API keys",
		Models: []ModelInfo{
			{ID: "command-a-03-2025", Name: "Command A", Tools: true, ContextWindow: 256_000},
			{ID: "command-a-reasoning-08-2025", Name: "Command A Reasoning", Tools: true, Reasoning: true, ContextWindow: 256_000},
			{ID: "command-r7b-12-2024", Name: "Command R7B", Tools: true, ContextWindow: 128_000},
		},
	},

	// Inference platforms: many open-weight models behind one key.
	{
		ID: "openrouter", Name: "OpenRouter", Kind: KindOpenAI, BaseURL: "https://openrouter.ai/api/v1", Group: GroupPlatforms,
		KeyHint: "openrouter.ai → Keys", Thinking: ThinkingOpenRouter,
		Headers: map[string]string{"HTTP-Referer": "https://github.com/mehran9675/pocket-agent", "X-Title": "Pocket Agent"},
	},
	{ID: "deepinfra", Name: "DeepInfra", Kind: KindOpenAI, BaseURL: "https://api.deepinfra.com/v1/openai", Group: GroupPlatforms, KeyHint: "deepinfra.com → Dashboard → API Keys", Thinking: ThinkingEffort},
	{ID: "together", Name: "Together AI", Kind: KindOpenAI, BaseURL: "https://api.together.ai/v1", Group: GroupPlatforms, KeyHint: "api.together.ai → Settings → API Keys"},
	{ID: "groq", Name: "Groq", Kind: KindOpenAI, BaseURL: "https://api.groq.com/openai/v1", Group: GroupPlatforms, KeyHint: "console.groq.com → API Keys", Thinking: ThinkingEffort},
	{ID: "fireworks", Name: "Fireworks AI", Kind: KindOpenAI, BaseURL: "https://api.fireworks.ai/inference/v1", Group: GroupPlatforms, KeyHint: "fireworks.ai → Settings → API Keys", Thinking: ThinkingEffort},
	{ID: "cerebras", Name: "Cerebras", Kind: KindOpenAI, BaseURL: "https://api.cerebras.ai/v1", Group: GroupPlatforms, KeyHint: "cloud.cerebras.ai → API Keys", Thinking: ThinkingEffort},
	{ID: "sambanova", Name: "SambaNova", Kind: KindOpenAI, BaseURL: "https://api.sambanova.ai/v1", Group: GroupPlatforms, KeyHint: "cloud.sambanova.ai → API Keys"},
	{ID: "huggingface", Name: "Hugging Face", Kind: KindOpenAI, BaseURL: "https://router.huggingface.co/v1", Group: GroupPlatforms, KeyHint: "huggingface.co → Settings → Access Tokens (Inference Providers permission)"},
	{ID: "nvidia", Name: "NVIDIA NIM", Kind: KindOpenAI, BaseURL: "https://integrate.api.nvidia.com/v1", Group: GroupPlatforms, KeyHint: "build.nvidia.com → Get API Key"},
	{ID: "novita", Name: "Novita AI", Kind: KindOpenAI, BaseURL: "https://api.novita.ai/v3/openai", Group: GroupPlatforms, KeyHint: "novita.ai → Key Management"},
	{ID: "hyperbolic", Name: "Hyperbolic", Kind: KindOpenAI, BaseURL: "https://api.hyperbolic.xyz/v1", Group: GroupPlatforms, KeyHint: "app.hyperbolic.xyz → Settings → API Key"},
	{ID: "vercel", Name: "Vercel AI Gateway", Kind: KindOpenAI, BaseURL: "https://ai-gateway.vercel.sh/v1", Group: GroupPlatforms, KeyHint: "vercel.com → AI Gateway → API Keys"},

	// Servers on the user's own network.
	{
		ID: "ollama", Name: "Ollama", Kind: KindOpenAI, BaseURL: "http://192.168.1.10:11434/v1", Group: GroupNetwork, Thinking: ThinkingEffort,
		KeyOptional: true, LAN: true, KeyHint: "Run `OLLAMA_HOST=0.0.0.0 ollama serve` on a computer on your network",
	},
	{
		ID: "lmstudio", Name: "LM Studio", Kind: KindOpenAI, BaseURL: "http://192.168.1.10:1234/v1", Group: GroupNetwork, Thinking: ThinkingEffort,
		KeyOptional: true, LAN: true, KeyHint: "Developer tab → Start server, and enable “Serve on local network”",
	},
	{
		ID: "llamacpp", Name: "llama.cpp server", Kind: KindOpenAI, BaseURL: "http://192.168.1.10:8080/v1", Group: GroupNetwork, Thinking: ThinkingTemplate,
		KeyOptional: true, LAN: true, KeyHint: "Run `llama-server -m model.gguf --host 0.0.0.0 --jinja` on a computer on your network",
	},
	{
		ID: "vllm", Name: "vLLM", Kind: KindOpenAI, BaseURL: "http://192.168.1.10:8000/v1", Group: GroupNetwork, Thinking: ThinkingTemplate,
		KeyOptional: true, LAN: true, KeyHint: "Run `vllm serve <model> --host 0.0.0.0`; add --api-key and enter it here if you set one",
	},
	{
		ID: "custom", Name: "Custom (OpenAI-compatible)", Kind: KindOpenAI, BaseURL: "https://", Group: GroupNetwork, Thinking: ThinkingTemplate,
		KeyOptional: true, KeyHint: "Any server speaking the Chat Completions API: LiteLLM, SGLang, TGI, a company gateway, ...",
	},
}

// Preset groups, in display order; "" is GroupCloud.
const (
	GroupCloud     = ""
	GroupPlatforms = "platforms"
	GroupNetwork   = "network"
)

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
			Thinking:      extra.Thinking,
		}
	}
}

func newCallID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "call_" + hex.EncodeToString(b[:])
}
