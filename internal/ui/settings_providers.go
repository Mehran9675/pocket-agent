package ui

import (
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) providersBody() widgets.El {
	groups := map[string][]widgets.El{}
	for _, pre := range llm.Presets {
		pre := pre
		cfg := a.settings.Provider(pre.ID)
		sub := tr("Not set up")
		var trail widgets.El
		if cfg != nil && cfg.Enabled {
			sub = tr("Connected")
			if len(cfg.Models) > 0 {
				sub = trn("Connected · {n} models", len(cfg.Models))
			}
			trail = widgets.Row(widgets.St{Gap: 8}, widgets.Dot(theme.T.Success, 8), widgets.Icon("chevron-right", 18, theme.T.Faint))
		} else if pre.LAN {
			sub = tr("Runs on a computer on your network")
		}
		groups[pre.Group] = append(groups[pre.Group], widgets.ListRow("prov/"+pre.ID, providerIcon(pre.ID), providerColor(pre.ID), pre.Name, sub, trail, func() {
			a.navigate("settings/provider/" + pre.ID)
		}))
	}
	kids := []widgets.El{
		widgets.Text(tr("Bring your own API keys. Keys stay in this device's secure storage and are sent only to the provider they belong to."), theme.SizeFootnote, theme.T.Muted),
	}
	for _, g := range []struct{ id, title string }{
		{llm.GroupCloud, tr("Model makers")},
		{llm.GroupPlatforms, tr("Inference platforms")},
		{llm.GroupNetwork, tr("Your network and custom servers")},
	} {
		if rows := groups[g.id]; len(rows) > 0 {
			kids = append(kids, widgets.SectionLabel(g.title), widgets.Group(rows...))
		}
	}
	return widgets.Col(widgets.St{Gap: 10, Width: -1}, kids...)
}
