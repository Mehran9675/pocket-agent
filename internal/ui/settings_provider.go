package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) providerTitle(id string) string {
	if p := a.settings.Provider(id); p != nil {
		return p.Name
	}
	if pre, ok := llm.PresetByID(id); ok {
		return pre.Name
	}
	return "Provider"
}

func (a *App) providerBody(id string) widgets.El {
	pre, _ := llm.PresetByID(id)
	cfg := a.settings.Provider(id)
	keyIn := a.input("prov/key/"+id, "API key", func() string {
		if cfg != nil {
			return cfg.APIKey
		}
		return ""
	}(), func(i *input.Input) { i.Password(true) })
	baseIn := a.input("prov/base/"+id, "Base URL", func() string {
		if cfg != nil && cfg.BaseURL != "" {
			return cfg.BaseURL
		}
		return pre.BaseURL
	}(), func(i *input.Input) { i.Keyboard(giode.KeyboardURL) })
	nameIn := a.input("prov/name/"+id, "Display name", func() string {
		if cfg != nil {
			return cfg.Name
		}
		return pre.Name
	}())
	showBase := pre.Kind == llm.KindOpenAI
	save := func() {
		c := a.settings.Provider(id)
		if c == nil {
			c = &store.ProviderConfig{ID: id, Preset: id}
			a.settings.Providers = append(a.settings.Providers, c)
		}
		c.Name = strings.TrimSpace(nameIn.Text())
		if c.Name == "" {
			c.Name = pre.Name
		}
		c.APIKey = strings.TrimSpace(keyIn.Text())
		if showBase {
			// Only a changed address is saved, so fixes to the preset's
			// address reach existing connections.
			c.BaseURL = strings.TrimSpace(baseIn.Text())
			if strings.TrimRight(c.BaseURL, "/") == strings.TrimRight(pre.BaseURL, "/") {
				c.BaseURL = ""
			}
		}
		c.Enabled = true
		a.saveSettings()
		a.refreshModels(c)
	}
	status := a.providerStatus[id]
	var statusEl widgets.El = giode.Box(widgets.St{})
	if a.refresh[id] {
		statusEl = widgets.Row(widgets.St{Gap: 8}, giode.Spinner(widgets.St{Width: 14, Height: 14, Color: theme.T.Primary}), widgets.Text("Connecting…", theme.SizeFootnote, theme.T.Muted))
	} else if status != "" {
		color, icon := theme.T.Success, "circle-check"
		if !strings.HasPrefix(status, "Connected") {
			color, icon = theme.T.Danger, "triangle-alert"
		}
		statusEl = widgets.Row(widgets.St{Gap: 8, Align: giode.AlignStart}, widgets.Icon(icon, 16, color), giode.Box(widgets.St{FlexGrow: 1}, widgets.Text(status, theme.SizeFootnote, color)))
	}
	formRow := func(label string, el widgets.El, hint string) widgets.El {
		kids := []widgets.El{widgets.Text(label, theme.SizeFootnote, theme.T.Text2, giode.FontWeightMedium), el}
		if hint != "" {
			kids = append(kids, widgets.Text(hint, theme.SizeCaption, theme.T.Muted))
		}
		return widgets.Col(widgets.St{Gap: 6, Width: -1}, kids...)
	}
	keyHint := pre.KeyHint
	if pre.KeyOptional {
		keyHint = "Optional for local servers. " + pre.KeyHint
	}
	kids := []widgets.El{
		widgets.Row(widgets.St{Gap: 14, Width: -1}, widgets.IconTile(providerIcon(id), providerColor(id), 48),
			widgets.Col(widgets.St{Gap: 2, FlexGrow: 1}, widgets.Text(pre.Name, theme.SizeTitle, theme.T.Text, giode.FontWeightSemiBold),
				widgets.Text(map[llm.Kind]string{llm.KindAnthropic: "Claude models via the Anthropic API", llm.KindGemini: "Gemini models via Google AI Studio", llm.KindOpenAI: "OpenAI-compatible API"}[pre.Kind], theme.SizeFootnote, theme.T.Muted))),
		widgets.Card(widgets.St{Gap: 16},
			formRow("API key", keyIn, keyHint),
			widgets.When(showBase, func() widgets.El {
				hint := ""
				if pre.LAN {
					hint = "Use your computer's IP address on the local network, e.g. http://192.168.1.20:11434/v1"
				}
				return formRow("Server URL", baseIn, hint)
			}),
			formRow("Display name", nameIn, ""),
			statusEl,
			widgets.Row(widgets.St{Gap: 10, Width: -1},
				widgets.Button("prov/save/"+id, ifStr(cfg != nil, "Save & refresh models", "Connect"), "plug", widgets.BtnPrimary, save, widgets.St{FlexGrow: 1}),
			),
		),
	}
	if cfg != nil {
		cfg := cfg
		if len(cfg.Models) > 0 {
			pinned := map[string]bool{}
			for _, m := range cfg.Pinned {
				pinned[m] = true
			}
			var rows []widgets.El
			models := append([]llm.ModelInfo(nil), cfg.Models...)
			sort.SliceStable(models, func(i, j int) bool { return pinned[models[i].ID] && !pinned[models[j].ID] })
			for _, m := range models {
				m := m
				star := "star"
				color := theme.T.Faint
				if pinned[m.ID] {
					color = theme.T.Warning
				}
				rows = append(rows, widgets.Row(widgets.St{Gap: 10, Padding: giode.Inset{Left: 14, Right: 6, Top: 4, Bottom: 4}, Width: -1},
					widgets.Col(widgets.St{Gap: 1, FlexGrow: 1}, widgets.Line(firstNonEmpty(m.Name, m.ID), theme.SizeCallout, theme.T.Text), widgets.When(m.Name != "" && m.Name != m.ID, func() widgets.El { return widgets.Line(m.ID, theme.SizeCaption, theme.T.Muted) })),
					widgets.IconButton("pin/"+id+"/"+m.ID, star, "Favorite", 17, color, func() {
						if pinned[m.ID] {
							for i, p := range cfg.Pinned {
								if p == m.ID {
									cfg.Pinned = append(cfg.Pinned[:i], cfg.Pinned[i+1:]...)
									break
								}
							}
						} else {
							cfg.Pinned = append(cfg.Pinned, m.ID)
						}
						a.saveSettings()
					})))
			}
			kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(fmt.Sprintf("Models · %d", len(cfg.Models))),
				widgets.Text("Star models to keep them at the top of the picker.", theme.SizeCaption, theme.T.Muted), widgets.Gap(0, 8),
				widgets.Card(widgets.St{Padding: giode.UniformInset(4), Gap: 0}, rows...)))
		}
		kids = append(kids, widgets.Group(
			widgets.ListRow("prov/enabled/"+id, "zap", theme.T.Primary, "Enabled", "Show this provider's models in the picker", widgets.Toggle("prov/en/"+id, cfg.Enabled, func(v bool) {
				cfg.Enabled = v
				a.saveSettings()
			}), nil),
			widgets.ListRow("prov/remove/"+id, "trash-2", theme.T.Danger, "Remove provider", "Deletes the stored key", nil, func() {
				a.ask("Remove "+cfg.Name+"?", "The API key is deleted from this device.", "Remove", true, func() {
					for i, p := range a.settings.Providers {
						if p == cfg {
							a.settings.Providers = append(a.settings.Providers[:i], a.settings.Providers[i+1:]...)
							break
						}
					}
					if a.settings.DefaultModel.Provider == id {
						a.settings.DefaultModel = store.ModelRef{}
						a.ensureDefaultModel()
					}
					a.dropInput("prov/key/" + id)
					a.saveSettings()
					a.back()
				})
			}),
		))
	}
	return widgets.Col(widgets.St{Gap: 18, Width: -1}, kids...)
}
