package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (v *chatView) modeBadge() widgets.El {
	a := v.a
	c := a.current
	switch c.Mode {
	case store.ModeAgent:
		if ag := a.agentByID(c.AgentID); ag != nil {
			return widgets.Row(widgets.St{Gap: 6, Padding: giode.SymmetricInset(10, 5), BorderRadius: 999, Background: theme.Tint(ag.Color, 0.14)},
				widgets.Icon(ag.Icon, 14, ag.Color), widgets.Text(agentName(ag), theme.SizeFootnote, ag.Color, giode.FontWeightMedium))
		}
	case store.ModeResearch:
		return widgets.Row(widgets.St{Gap: 6, Padding: giode.SymmetricInset(10, 5), BorderRadius: 999, Background: theme.Tint("#38bdf8", 0.14)},
			widgets.Icon("telescope", 14, "#38bdf8"), widgets.Text(tr("Deep research"), theme.SizeFootnote, "#38bdf8", giode.FontWeightMedium))
	}
	return giode.Box(widgets.St{})
}
