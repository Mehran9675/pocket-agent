package ui

func providerIcon(preset string) string {
	switch preset {
	case "ollama", "lmstudio", "llamacpp", "vllm":
		return "server"
	case "custom":
		return "plug"
	}
	return "cloud"
}

func providerColor(preset string) string {
	switch preset {
	case "anthropic":
		return "#d97757"
	case "openai":
		return "#10a37f"
	case "gemini":
		return "#4285f4"
	case "openrouter":
		return "#6467f2"
	case "groq":
		return "#f55036"
	case "mistral":
		return "#fa520f"
	case "deepseek":
		return "#4d6bfe"
	case "xai":
		return "#9ca3af"
	case "together":
		return "#0f6fff"
	case "ollama", "lmstudio", "llamacpp", "vllm":
		return "#f59e0b"
	case "deepinfra":
		return "#5b6cf9"
	case "fireworks":
		return "#7c3aed"
	case "cerebras":
		return "#f15a29"
	case "sambanova":
		return "#ee7624"
	case "huggingface":
		return "#ffb000"
	case "nvidia":
		return "#76b900"
	case "perplexity":
		return "#20b8cd"
	case "cohere":
		return "#39594d"
	case "moonshot":
		return "#1f6feb"
	case "zai":
		return "#3b82f6"
	case "qwen":
		return "#615ced"
	case "minimax":
		return "#e5484d"
	case "novita":
		return "#22c55e"
	case "hyperbolic":
		return "#a855f7"
	case "vercel":
		return "#e5e7eb"
	}
	return "#8b93a7"
}
