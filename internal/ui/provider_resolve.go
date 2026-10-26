package ui

import (
	"errors"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// resolve builds the provider for a model reference.
func (a *App) resolve(ref store.ModelRef) (llm.Provider, llm.ModelInfo, error) {
	info := llm.ModelInfo{ID: ref.Model, Tools: true}
	if ref.IsZero() {
		return nil, info, errors.New("choose a model first")
	}
	if ref.Provider == store.LocalProviderID {
		info.ContextWindow = a.settings.Local.ContextSize
		for _, m := range a.models.Installed() {
			if m.File == ref.Model {
				info.Name, info.Reasoning = m.Name, m.Reasoning
			}
		}
		return a.localProv, info, nil
	}
	p := a.settings.Provider(ref.Provider)
	if p == nil {
		return nil, info, errors.New("this model's provider was removed - pick another model")
	}
	pre, _ := llm.PresetByID(p.Preset)
	info.Vision = pre.Kind != llm.KindOpenAI || p.Preset == "openai" || p.Preset == "openrouter" || p.Preset == "xai"
	for _, m := range append(append([]llm.ModelInfo{}, pre.Models...), p.Models...) {
		if m.ID == ref.Model {
			if m.Vision {
				info.Vision = true
			}
			info.Reasoning = info.Reasoning || m.Reasoning
			if m.ContextWindow > 0 {
				info.ContextWindow = m.ContextWindow
			}
		}
	}
	base := p.BaseURL
	if base == "" {
		base = pre.BaseURL
	}
	return llm.New(pre.Kind, p.Name, base, p.APIKey, pre), info, nil
}
