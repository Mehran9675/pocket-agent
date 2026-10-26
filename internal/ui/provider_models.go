package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mehran9675/pocket-agent/internal/agent"
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// refreshModels fetches a provider's model list.
func (a *App) refreshModels(p *store.ProviderConfig) {
	if a.refresh[p.ID] {
		return
	}
	a.refresh[p.ID] = true
	a.providerStatus[p.ID] = ""
	ref := store.ModelRef{Provider: p.ID, Model: "x"}
	prov, _, err := a.resolve(ref)
	if err != nil {
		a.refresh[p.ID] = false
		a.providerStatus[p.ID] = err.Error()
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		models, err := prov.Models(ctx)
		a.Post(func() {
			a.refresh[p.ID] = false
			if err != nil {
				a.providerStatus[p.ID] = agent.FriendlyError(err)
				return
			}
			models = filterChatModels(p.Preset, models)
			p.Models = models
			a.providerStatus[p.ID] = fmt.Sprintf("Connected · %d models", len(models))
			if a.settings.DefaultModel.IsZero() && len(models) > 0 {
				a.settings.DefaultModel = store.ModelRef{Provider: p.ID, Model: preferredModel(p.Preset, models)}
				a.draft.model = a.settings.DefaultModel
			}
			a.saveSettings()
		})
	}()
}

// filterChatModels drops embedding, audio and image models.
func filterChatModels(preset string, ms []llm.ModelInfo) []llm.ModelInfo {
	var out []llm.ModelInfo
	for _, m := range ms {
		id := strings.ToLower(m.ID)
		skip := false
		for _, bad := range []string{"embed", "whisper", "tts", "dall-e", "moderation", "audio", "realtime", "transcribe", "image", "rerank", "guard", "davinci", "babbage", "search"} {
			if strings.Contains(id, bad) {
				skip = true
			}
		}
		if !skip {
			out = append(out, m)
		}
	}
	return out
}

// preferredModel picks a sensible default from a fresh list.
func preferredModel(preset string, ms []llm.ModelInfo) string {
	prefs := map[string][]string{
		"anthropic": {"claude-opus-5-5", "claude-sonnet-5-5"},
		"openai":    {"gpt-5", "gpt-4.1", "gpt-4o"},
		"gemini":    {"gemini-2.5-pro", "gemini-2.5-flash"},
	}
	for _, want := range prefs[preset] {
		for _, m := range ms {
			if m.ID == want {
				return m.ID
			}
		}
	}
	return ms[0].ID
}
