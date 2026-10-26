package ui

import (
	"fmt"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) providersBody() widgets.El {
	var rows []widgets.El
	for _, pre := range llm.Presets {
		pre := pre
		cfg := a.settings.Provider(pre.ID)
		sub := "Not set up"
		var trail widgets.El
		if cfg != nil && cfg.Enabled {
			sub = "Connected"
			if len(cfg.Models) > 0 {
				sub = fmt.Sprintf("Connected · %d models", len(cfg.Models))
			}
			trail = widgets.Row(widgets.St{Gap: 8}, widgets.Dot(theme.T.Success, 8), widgets.Icon("chevron-right", 18, theme.T.Faint))
		} else if pre.LAN {
			sub = "Runs on a computer on your network"
		}
		rows = append(rows, widgets.ListRow("prov/"+pre.ID, providerIcon(pre.ID), providerColor(pre.ID), pre.Name, sub, trail, func() {
			a.navigate("settings/provider/" + pre.ID)
		}))
	}
	return widgets.Col(widgets.St{Gap: 14, Width: -1},
		widgets.Text("Bring your own API keys. Keys are stored only on this device and sent only to the provider they belong to.", theme.SizeFootnote, theme.T.Muted),
		widgets.Group(rows...),
	)
}
