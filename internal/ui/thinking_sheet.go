package ui

import (
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// modelThinks reports whether ref is a reasoning model whose thinking
// the app can turn down or off.
func (a *App) modelThinks(ref store.ModelRef) bool {
	if ref.IsZero() {
		return false
	}
	if ref.Provider == store.LocalProviderID {
		for _, m := range a.models.Installed() {
			if m.File == ref.Model {
				return m.Reasoning
			}
		}
		return false
	}
	p := a.settings.Provider(ref.Provider)
	if p == nil {
		return false
	}
	pre, _ := llm.PresetByID(p.Preset)
	if !llm.CanControlThinking(pre.Kind, pre) {
		return false
	}
	for _, list := range [][]llm.ModelInfo{pre.Models, p.Models} {
		for _, m := range list {
			if m.ID == ref.Model && m.Reasoning {
				return true
			}
		}
	}
	return llm.LooksLikeReasoning(ref.Model)
}

// thinking is the level for the open chat, or for the next one.
func (a *App) thinking() llm.Thinking {
	if a.current != nil {
		return a.current.Thinking
	}
	return a.settings.Thinking
}

func (a *App) openThinkingSheet() {
	cur := a.thinking()
	set := func(t llm.Thinking) {
		a.closeSheet()
		if c := a.current; c != nil {
			c.Thinking = t
			c.Version++
			a.st.SaveConversation(c)
		}
		a.settings.Thinking = t
		a.saveSettings()
	}
	row := func(t llm.Thinking, sub string) widgets.El {
		var check widgets.El
		if t == cur {
			check = widgets.Icon("check", 18, theme.T.Primary)
		}
		return widgets.ListRow("think/"+string(t)+"x", "brain", "#f59e0b", tr(t.Label()), sub, check, func() { set(t) })
	}
	a.openSheet("thinking", tr("Thinking"), func() widgets.El {
		return widgets.Col(widgets.St{Gap: 10, Width: -1},
			widgets.Group(
				row(llm.ThinkingAuto, tr("The model decides how long to think")),
				row(llm.ThinkingOff, tr("Answer right away - fastest and cheapest")),
				row(llm.ThinkingLow, tr("A little reasoning for everyday questions")),
				row(llm.ThinkingMedium, tr("Balanced depth and speed")),
				row(llm.ThinkingHigh, tr("Thinks longest - for hard problems, math and code")),
			),
			widgets.Text(tr("Applies to models that reason. Some providers only offer some levels; the closest one is used."), theme.SizeFootnote, theme.T.Muted))
	})
}

// thinkingChip is the composer's thinking control, shown for reasoning
// models.
func (a *App) thinkingChip(compact bool) widgets.El {
	t := a.thinking()
	label := tr(t.Label())
	if !compact {
		label = tr("Thinking: {level}", "level", label)
	}
	return widgets.Chip("composer/think", label, "brain", t != llm.ThinkingAuto, a.openThinkingSheet)
}
