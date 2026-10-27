package ui

import (
	"fmt"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) providersBody() widgets.El {
	groups := map[string][]widgets.El{}
	for _, pre := range llm.Presets {
		pre := pre
		cfg := a.settings.Provider(pre.ID)
		sub := "Not set up"
		var trail widgets.El
		if cfg != nil && cfg.Enabled {
			sub = "Connected"
			if len(cfg.Models) > 0 {
				sub = fmt.Sprintf("Connected · %d model%s", len(cfg.Models), ifStr(len(cfg.Models) == 1, "", "s"))
			}
			trail = widgets.Row(widgets.St{Gap: 8}, widgets.Dot(theme.T.Success, 8), widgets.Icon("chevron-right", 18, theme.T.Faint))
		} else if pre.LAN {
			sub = "Runs on a computer on your network"
		}
		groups[pre.Group] = append(groups[pre.Group], widgets.ListRow("prov/"+pre.ID, providerIcon(pre.ID), providerColor(pre.ID), pre.Name, sub, trail, func() {
			a.navigate("settings/provider/" + pre.ID)
		}))
	}
	kids := []widgets.El{
		widgets.Text("Bring your own API keys. Keys stay in this device's secure storage and are sent only to the provider they belong to.", theme.SizeFootnote, theme.T.Muted),
	}
	for _, g := range []struct{ id, title string }{
		{llm.GroupCloud, "Model makers"},
		{llm.GroupPlatforms, "Inference platforms"},
		{llm.GroupNetwork, "Your network and custom servers"},
	} {
		if rows := groups[g.id]; len(rows) > 0 {
			kids = append(kids, widgets.SectionLabel(g.title), widgets.Group(rows...))
		}
	}
	return widgets.Col(widgets.St{Gap: 10, Width: -1}, kids...)
}
