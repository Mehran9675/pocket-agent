package ui

func providerIcon(preset string) string {
	switch preset {
	case "ollama", "lmstudio":
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
	case "ollama", "lmstudio":
		return "#f59e0b"
	}
	return "#8b93a7"
}
