package ui

import (
	"strings"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// modelLabel names a model reference for display.
func (a *App) modelLabel(ref store.ModelRef) (name, provider string) {
	if ref.IsZero() {
		return tr("Choose a model"), ""
	}
	if ref.Provider == store.LocalProviderID {
		for _, m := range a.models.Installed() {
			if m.File == ref.Model {
				return m.Name, tr("On device")
			}
		}
		return strings.TrimSuffix(ref.Model, ".gguf"), tr("On device")
	}
	p := a.settings.Provider(ref.Provider)
	if p == nil {
		return ref.Model, ""
	}
	for _, m := range p.Models {
		if m.ID == ref.Model && m.Name != "" {
			return m.Name, p.Name
		}
	}
	if pre, ok := llm.PresetByID(p.Preset); ok {
		for _, m := range pre.Models {
			if m.ID == ref.Model {
				return m.Name, p.Name
			}
		}
	}
	return ref.Model, p.Name
}

// currentModel is the model the next message uses.
func (a *App) currentModel() store.ModelRef {
	if a.current != nil && !a.current.Model.IsZero() {
		return a.current.Model
	}
	if !a.draft.model.IsZero() {
		return a.draft.model
	}
	return a.settings.DefaultModel
}

func (a *App) setModel(ref store.ModelRef) {
	if a.current != nil {
		a.current.Model = ref
		a.current.Touch()
		a.st.SaveConversation(a.current)
	}
	a.draft.model = ref
	if a.settings.DefaultModel.IsZero() {
		a.settings.DefaultModel = ref
		a.saveSettings()
	}
}

// ensureDefaultModel picks a model when none is set yet.
func (a *App) ensureDefaultModel() {
	if !a.settings.DefaultModel.IsZero() {
		return
	}
	for _, p := range a.settings.Providers {
		if p.Enabled && len(p.Models) > 0 {
			a.settings.DefaultModel = store.ModelRef{Provider: p.ID, Model: p.Models[0].ID}
			a.draft.model = a.settings.DefaultModel
			return
		}
	}
	if inst := a.models.Installed(); len(inst) > 0 && a.engine.Available() {
		a.settings.DefaultModel = store.ModelRef{Provider: store.LocalProviderID, Model: inst[0].File}
		a.draft.model = a.settings.DefaultModel
	}
}

// hasAnyModel reports whether anything is set up to chat with.
func (a *App) hasAnyModel() bool {
	for _, p := range a.settings.Providers {
		if !p.Enabled {
			continue
		}
		if pre, _ := llm.PresetByID(p.Preset); p.APIKey != "" || pre.KeyOptional {
			return true
		}
	}
	return a.engine.Available() && len(a.models.Installed()) > 0
}
