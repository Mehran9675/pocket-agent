package ui

import (
	"fmt"
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// openModelSheet lets the user pick the model for this chat.
func (a *App) openModelSheet() {
	search := a.input("models/search", "Search models", "")
	search.SetText("")
	for _, p := range a.settings.Providers {
		if p.Enabled && len(p.Models) == 0 && p.APIKey != "" {
			a.refreshModels(p)
		}
	}
	sh := a.openSheet("models", "Choose a model", func() widgets.El { return a.modelList(search.Text()) })
	sh.width = 560
	sh.footer = func() widgets.El {
		return widgets.Row(widgets.St{Gap: 10, Width: -1},
			widgets.Button("models/providers", "Providers", "key-round", widgets.BtnSecondary, func() {
				a.closeSheet()
				a.navigate("settings/providers")
			}),
			widgets.Button("models/library", "On-device", "hard-drive", widgets.BtnSecondary, func() {
				a.closeSheet()
				a.navigate("library")
			}),
		)
	}
}

func (a *App) modelList(query string) widgets.El {
	q := strings.ToLower(strings.TrimSpace(query))
	cur := a.currentModel()
	search := a.input("models/search", "Search models", "")
	kids := []widgets.El{giode.Box(widgets.St{Width: -1, Padding: giode.Inset{Bottom: 4}}, search)}

	pick := func(ref store.ModelRef) {
		a.setModel(ref)
		a.closeSheet()
	}
	modelRow := func(ref store.ModelRef, name, sub string, badges ...widgets.El) widgets.El {
		selected := ref == cur
		return widgets.Tap("models/"+ref.Provider+"/"+ref.Model, name, func(s giode.PressState) widgets.El {
			bg := widgets.HoverBg(s, "transparent")
			if selected {
				bg = theme.T.PrimarySoft
			}
			right := []widgets.El{}
			right = append(right, badges...)
			if selected {
				right = append(right, widgets.Icon("check", 18, theme.T.Primary))
			}
			texts := []widgets.El{widgets.Line(name, theme.SizeCallout, theme.T.Text, giode.FontWeightMedium)}
			if sub != "" {
				texts = append(texts, widgets.Line(sub, theme.SizeCaption, theme.T.Muted))
			}
			return widgets.Row(widgets.St{Gap: 10, Padding: giode.SymmetricInset(12, 10), BorderRadius: 12, Background: bg, Width: -1},
				append([]widgets.El{widgets.Col(widgets.St{Gap: 2, FlexGrow: 1}, texts...)}, right...)...)
		}, func() { pick(ref) })
	}
	match := func(s ...string) bool {
		if q == "" {
			return true
		}
		for _, x := range s {
			if strings.Contains(strings.ToLower(x), q) {
				return true
			}
		}
		return false
	}

	// On-device
	var localRows []widgets.El
	if a.engine.Available() {
		for _, m := range a.models.Installed() {
			if !match(m.Name, m.File) {
				continue
			}
			var bs []widgets.El
			if m.Reasoning {
				bs = append(bs, widgets.Badge("Thinking", theme.T.Text2, theme.T.Surface2))
			}
			localRows = append(localRows, modelRow(store.ModelRef{Provider: store.LocalProviderID, Model: m.File}, m.Name, tools.HumanBytes(m.Size)+" · private, offline", bs...))
		}
	}
	if len(localRows) > 0 || q == "" {
		kids = append(kids, providerHeader("cpu", "#34d399", "On this device", ""))
		if len(localRows) == 0 {
			sub := "Download a model to chat offline"
			if !a.engine.Available() {
				sub = "Not available on this device"
			}
			localRows = append(localRows, widgets.ListRow("models/getlocal", "download", "#34d399", "Get an on-device model", sub, nil, func() {
				a.closeSheet()
				a.navigate("library")
			}))
		}
		kids = append(kids, widgets.Col(widgets.St{Gap: 2, Width: -1}, localRows...))
	}

	for _, p := range a.settings.Providers {
		if !p.Enabled {
			continue
		}
		p := p
		pre, _ := llm.PresetByID(p.Preset)
		models := p.Models
		if len(models) == 0 {
			models = pre.Models
		}
		var rows []widgets.El
		pinned := map[string]bool{}
		for _, id := range p.Pinned {
			pinned[id] = true
		}
		// Pinned first, then the rest; long lists are capped unless searching.
		ordered := make([]llm.ModelInfo, 0, len(models))
		for _, m := range models {
			if pinned[m.ID] {
				ordered = append(ordered, m)
			}
		}
		for _, m := range models {
			if !pinned[m.ID] {
				ordered = append(ordered, m)
			}
		}
		shown := 0
		for _, m := range ordered {
			name := m.Name
			if name == "" {
				name = m.ID
			}
			if !match(name, m.ID, p.Name) {
				continue
			}
			if q == "" && shown >= 12 && !pinned[m.ID] && cur.Model != m.ID {
				continue
			}
			shown++
			sub := m.ID
			if sub == name {
				sub = ""
			}
			if m.ContextWindow > 0 {
				if sub != "" {
					sub += " · "
				}
				sub += fmt.Sprintf("%s context", compactNum(m.ContextWindow))
			}
			var bs []widgets.El
			if pinned[m.ID] {
				bs = append(bs, widgets.Icon("star", 14, theme.T.Warning))
			}
			rows = append(rows, modelRow(store.ModelRef{Provider: p.ID, Model: m.ID}, name, sub, bs...))
		}
		if len(rows) == 0 && q != "" {
			continue
		}
		status := ""
		if a.refresh[p.ID] {
			status = "Loading…"
		} else if len(p.Models) > len(rows) && q == "" {
			status = fmt.Sprintf("%d models · search to see all", len(p.Models))
		}
		kids = append(kids, providerHeader(providerIcon(p.Preset), providerColor(p.Preset), p.Name, status))
		if len(rows) == 0 {
			rows = append(rows, giode.Box(widgets.St{Padding: giode.SymmetricInset(12, 8)}, widgets.Text("No models yet - check the provider's settings.", theme.SizeFootnote, theme.T.Muted)))
		}
		kids = append(kids, widgets.Col(widgets.St{Gap: 2, Width: -1}, rows...))
	}
	if len(a.settings.Providers) == 0 {
		kids = append(kids, widgets.Gap(0, 8), widgets.Card(widgets.St{},
			widgets.Text("No cloud providers yet", theme.SizeCallout, theme.T.Text, giode.FontWeightBold),
			widgets.Text("Add an API key for Anthropic, OpenAI, Google, OpenRouter and others to use their models here.", theme.SizeFootnote, theme.T.Muted),
			widgets.Button("models/addprov", "Add a provider", "plus", widgets.BtnPrimary, func() {
				a.closeSheet()
				a.navigate("settings/providers")
			})))
	}
	return widgets.Col(widgets.St{Gap: 6, Width: -1}, kids...)
}
