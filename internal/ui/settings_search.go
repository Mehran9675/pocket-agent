package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) searchBody() widgets.El {
	cfg := &a.settings.Search
	engines := []struct {
		id         tools.SearchEngine
		name, desc string
	}{
		{tools.SearchDuckDuckGo, "DuckDuckGo", "No key needed. Good default; may rate-limit heavy use."},
		{tools.SearchBrave, "Brave Search", "Independent index with a free API tier. Needs an API key."},
		{tools.SearchTavily, "Tavily", "Search built for AI agents. Needs an API key."},
		{tools.SearchSearXNG, "SearXNG", "Your own private metasearch instance."},
	}
	var rows []widgets.El
	for _, e := range engines {
		e := e
		var tr widgets.El
		if cfg.Engine == e.id || (cfg.Engine == "" && e.id == tools.SearchDuckDuckGo) {
			tr = widgets.Icon("check", 18, theme.T.Primary)
		}
		rows = append(rows, widgets.ListRow("eng/"+string(e.id), "globe", "#38bdf8", e.name, e.desc, tr, func() {
			cfg.Engine = e.id
			a.saveSettings()
		}))
	}
	kids := []widgets.El{
		widgets.Text("Used by the web_search tool in chat, by agents and by deep research.", theme.SizeFootnote, theme.T.Muted),
		widgets.Group(rows...),
	}
	switch cfg.Engine {
	case tools.SearchBrave, tools.SearchTavily:
		in := a.input("search/key/"+string(cfg.Engine), "API key", cfg.APIKey, func(i *input.Input) { i.Password(true) })
		kids = append(kids, widgets.Card(widgets.St{},
			widgets.Text("API key", theme.SizeFootnote, theme.T.Text2, giode.FontWeightMedium), in,
			widgets.Button("search/savekey", "Save", "", widgets.BtnPrimary, func() {
				cfg.APIKey = strings.TrimSpace(in.Text())
				a.saveSettings()
				a.toast(giode.ToastSuccess, "Saved")
			})))
	case tools.SearchSearXNG:
		in := a.input("search/base", "https://searx.example.org", cfg.BaseURL, func(i *input.Input) { i.Keyboard(giode.KeyboardURL) })
		kids = append(kids, widgets.Card(widgets.St{},
			widgets.Text("Instance URL", theme.SizeFootnote, theme.T.Text2, giode.FontWeightMedium), in,
			widgets.Text("The instance must allow the JSON output format.", theme.SizeCaption, theme.T.Muted),
			widgets.Button("search/savebase", "Save", "", widgets.BtnPrimary, func() {
				cfg.BaseURL = strings.TrimSpace(in.Text())
				a.saveSettings()
				a.toast(giode.ToastSuccess, "Saved")
			})))
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1}, kids...)
}
