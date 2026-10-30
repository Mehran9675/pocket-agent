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
		{tools.SearchDuckDuckGo, "DuckDuckGo", tr("No key needed. Good default; may rate-limit heavy use.")},
		{tools.SearchBrave, "Brave Search", tr("Independent index with a free API tier. Needs an API key.")},
		{tools.SearchTavily, "Tavily", tr("Search built for AI agents. Needs an API key.")},
		{tools.SearchSearXNG, "SearXNG", tr("Your own private metasearch instance.")},
	}
	var rows []widgets.El
	for _, e := range engines {
		e := e
		var check widgets.El
		if cfg.Engine == e.id || (cfg.Engine == "" && e.id == tools.SearchDuckDuckGo) {
			check = widgets.Icon("check", 18, theme.T.Primary)
		}
		rows = append(rows, widgets.ListRow("eng/"+string(e.id), "globe", "#38bdf8", e.name, e.desc, check, func() {
			cfg.Engine = e.id
			a.saveSettings()
		}))
	}
	kids := []widgets.El{
		widgets.Text(tr("Used by the web_search tool in chat, by agents and by deep research."), theme.SizeFootnote, theme.T.Muted),
		widgets.Group(rows...),
	}
	switch cfg.Engine {
	case tools.SearchBrave, tools.SearchTavily:
		in := a.input("search/key/"+string(cfg.Engine), tr("API key"), cfg.APIKey, func(i *input.Input) { i.Password(true) })
		kids = append(kids, widgets.Card(widgets.St{},
			widgets.Text(tr("API key"), theme.SizeFootnote, theme.T.Text2, giode.FontWeightMedium), in,
			widgets.Button("search/savekey", tr("Save"), "", widgets.BtnPrimary, func() {
				cfg.APIKey = strings.TrimSpace(in.Text())
				a.saveSettings()
				a.toast(giode.ToastSuccess, tr("Saved"))
			})))
	case tools.SearchSearXNG:
		in := a.input("search/base", "https://searx.example.org", cfg.BaseURL, func(i *input.Input) { i.Keyboard(giode.KeyboardURL) })
		kids = append(kids, widgets.Card(widgets.St{},
			widgets.Text(tr("Instance URL"), theme.SizeFootnote, theme.T.Text2, giode.FontWeightMedium), in,
			widgets.Text(tr("The instance must allow the JSON output format."), theme.SizeCaption, theme.T.Muted),
			widgets.Button("search/savebase", tr("Save"), "", widgets.BtnPrimary, func() {
				cfg.BaseURL = strings.TrimSpace(in.Text())
				a.saveSettings()
				a.toast(giode.ToastSuccess, tr("Saved"))
			})))
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1}, kids...)
}
