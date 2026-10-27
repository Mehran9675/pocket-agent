package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) settingsHome() widgets.El {
	name, prov := a.modelLabel(a.settings.DefaultModel)
	connected := 0
	for _, p := range a.settings.Providers {
		if p.Enabled {
			connected++
		}
	}
	themeIdx := map[store.Theme]int{"system": 0, store.ThemeLight: 1, store.ThemeDark: 2}[a.settings.Theme]
	engines := map[tools.SearchEngine]string{tools.SearchDuckDuckGo: "DuckDuckGo", tools.SearchBrave: "Brave Search", tools.SearchTavily: "Tavily", tools.SearchSearXNG: "SearXNG"}
	mcpCount := len(a.settings.MCPServers)

	var swatches []widgets.El
	for _, ac := range theme.Accents {
		ac := ac
		swatches = append(swatches, widgets.Tap("accent/"+ac.ID, ac.Name, func(s giode.PressState) widgets.El {
			c := ac.Dark
			if !theme.T.Dark {
				c = ac.Light
			}
			border := "transparent"
			if a.settings.Accent == ac.ID {
				border = theme.T.Text
			}
			return giode.Box(widgets.St{Width: 34, Height: 34, BorderRadius: 17, BorderWidth: 2, BorderColor: border, Padding: giode.UniformInset(3)},
				giode.Box(widgets.St{Width: -1, Height: -1, BorderRadius: 14, Background: c}))
		}, func() {
			a.settings.Accent = ac.ID
			a.saveSettings()
			a.Theme()
		}))
	}
	return widgets.Col(widgets.St{Gap: 18, Width: -1},
		widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel("Models"), widgets.Group(
			widgets.ListRow("s/default", "sparkles", theme.T.Primary, "Default model", name+ifStr(prov != "", " · "+prov, ""), nil, a.openDefaultModelSheet),
			widgets.ListRow("s/providers", "key-round", "#d97757", "Cloud providers", fmt.Sprintf("%d connected", connected), nil, func() { a.navigate("settings/providers") }),
			widgets.ListRow("s/library", "hard-drive", "#34d399", "On-device models", fmt.Sprintf("%d installed", len(a.models.Installed())), nil, func() { a.navigate("library") }),
			widgets.ListRow("s/engine", "cpu", "#34d399", "On-device engine", engineSummary(a), nil, func() { a.navigate("settings/local") }),
		)),
		widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel("Capabilities"), widgets.Group(
			widgets.ListRow("s/search", "globe", "#38bdf8", "Web search", engines[a.settings.Search.Engine], nil, func() { a.navigate("settings/search") }),
			widgets.ListRow("s/tools", "wrench", "#a78bfa", "Tools & permissions", fmt.Sprintf("%d tools in chat", len(a.settings.ChatTools)), nil, func() { a.navigate("settings/tools") }),
			widgets.ListRow("s/mcp", "plug", "#f472b6", "MCP servers", fmt.Sprintf("%d configured", mcpCount), nil, func() { a.navigate("settings/mcp") }),
			widgets.ListRow("s/agents", "bot", "#7c83ff", "Agents", fmt.Sprintf("%d agents", len(a.agents)), nil, func() { a.navigate("agents") }),
		)),
		widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel("Personalization"), widgets.Group(
			widgets.ListRow("s/instructions", "pen-line", "#f59e0b", "Custom instructions", ifStr(a.settings.Instructions == "", "Not set", "On"), nil, func() { a.navigate("settings/instructions") }),
			widgets.ListRow("s/memory", "brain", "#f59e0b", "Memory", fmt.Sprintf("%d memories", len(a.st.All())), nil, func() { a.navigate("settings/memory") }),
			widgets.ListRow("s/reasoning", "lightbulb", "#f59e0b", "Show thinking", "Display reasoning from models that think", widgets.Toggle("t/reasoning", a.settings.ShowReasoning, func(v bool) {
				a.settings.ShowReasoning = v
				a.saveSettings()
			}), nil),
		)),
		widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel("Appearance"), widgets.Card(widgets.St{Gap: 16},
			widgets.Col(widgets.St{Gap: 8, Width: -1}, widgets.Text("Theme", theme.SizeCallout, theme.T.Text, giode.FontWeightMedium),
				widgets.Segmented("theme", []string{"System", "Light", "Dark"}, []string{"smartphone", "sun", "moon"}, themeIdx, func(i int) {
					a.settings.Theme = []store.Theme{"system", store.ThemeLight, store.ThemeDark}[i]
					a.saveSettings()
					a.Theme()
				})),
			widgets.Col(widgets.St{Gap: 10, Width: -1}, widgets.Text("Accent color", theme.SizeCallout, theme.T.Text, giode.FontWeightMedium),
				giode.Row(widgets.St{Gap: 10, Wrap: giode.WrapWrap}, swatches...)),
			widgets.Col(widgets.St{Gap: 8, Width: -1}, widgets.Text("Message text size", theme.SizeCallout, theme.T.Text, giode.FontWeightMedium),
				widgets.Segmented("fontscale", []string{"Small", "Default", "Large"}, nil, fontScaleIdx(a.settings.FontScale), func(i int) {
					a.settings.FontScale = []float64{0.92, 1, 1.12}[i]
					a.saveSettings()
					giode.InvalidateCaches()
				})),
		)),
		widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel("Data"), widgets.Group(
			widgets.ListRow("s/exportall", "download", "#34d399", "Export all conversations", "Save everything as one Markdown file", nil, a.exportAll),
			widgets.ListRow("s/deleteall", "trash-2", theme.T.Danger, "Delete all conversations", "Removes chats, attachments and agent files", nil, func() {
				a.ask("Delete all conversations?", "This permanently removes every conversation, attachment and workspace file on this device. Settings and models are kept.", "Delete all", true, func() {
					for _, c := range a.convs {
						a.stop(c)
					}
					a.convs = nil
					a.current = nil
					a.st.DeleteAll()
					a.toast(giode.ToastSuccess, "All conversations deleted")
				})
			}),
		)),
		widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel("Pocket Agent"), widgets.Group(
			widgets.ListRow("s/about", "info", theme.T.Muted, "About", "Version "+Version, nil, func() { a.navigate("settings/about") }),
		)),
	)
}

func fontScaleIdx(f float64) int {
	switch {
	case f == 0 || (f > 0.95 && f < 1.05):
		return 1
	case f < 1:
		return 0
	}
	return 2
}

func engineSummary(a *App) string {
	if !a.engine.Available() {
		return a.engineLabel()
	}
	if a.settings.Local.ContextSize < 0 {
		return a.engineLabel() + " · model's full context"
	}
	return fmt.Sprintf("%s · %dK context", a.engineLabel(), a.settings.Local.ContextSize/1024)
}
