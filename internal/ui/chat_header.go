package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (v *chatView) header(compact bool) widgets.El {
	a := v.a
	ref := a.currentModel()
	name, provider := a.modelLabel(ref)
	modelPill := widgets.Tap("header/model", tr("Choose model"), func(s giode.PressState) widgets.El {
		lead := widgets.Icon("cloud", 15, theme.T.Muted)
		if ref.Provider == store.LocalProviderID {
			lead = widgets.Icon("cpu", 15, theme.T.Success)
		}
		if ref.IsZero() {
			lead = widgets.Icon("sparkles", 15, theme.T.Primary)
		}
		kids := []widgets.El{lead, widgets.Line(name, theme.SizeCallout, theme.T.Text, giode.FontWeightSemiBold)}
		if provider != "" && !compact {
			kids = append(kids, widgets.Line(provider, theme.SizeFootnote, theme.T.Muted))
		}
		kids = append(kids, widgets.Icon("chevron-down", 15, theme.T.Muted))
		return widgets.Row(widgets.St{Gap: 7, Padding: giode.SymmetricInset(12, 8), BorderRadius: 12, Background: widgets.HoverBg(s, "transparent"), MaxWidth: 360}, kids...)
	}, a.openModelSheet)

	var left, right []widgets.El
	if compact {
		left = append(left, widgets.IconButton("header/menu", "panel-left", tr("Conversations"), 21, theme.T.Text, func() { a.drawerOpen = true }))
	}
	left = append(left, modelPill)
	if c := a.current; c != nil && len(c.Messages) > 0 {
		right = append(right, widgets.IconButton("header/more", "ellipsis", tr("Conversation options"), 20, theme.T.Text2, func() { a.openConversationMenu(c) }))
	}
	if compact || a.current != nil {
		right = append(right, widgets.IconButton("header/new", "square-pen", tr("New chat"), 20, theme.T.Text, func() { a.newChat(store.ModeChat, "") }))
	}
	kids := append(left, widgets.Spacer())
	if !compact && a.current != nil {
		kids = append(kids, v.modeBadge())
	}
	kids = append(kids, right...)
	return widgets.Row(widgets.St{Height: 56, Width: -1, Padding: giode.SymmetricInset(8, 0), Gap: 2}, kids...)
}
