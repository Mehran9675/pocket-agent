package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/swipeable"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (s *sidebarView) convRow(c *store.Conversation) widgets.El {
	a := s.a
	active := a.current == c && a.route == ""
	running := a.isRunning(c)
	row := widgets.TapAlt("side/conv/"+c.ID, c.Title, func(st giode.PressState) widgets.El {
		base := "transparent"
		if isMobile() {
			base = theme.T.Sidebar // opaque over the swipe actions
		}
		bg := widgets.HoverBg(st, base)
		if active {
			bg = theme.T.Hover
		}
		icon := "message-square"
		color := theme.T.Muted
		switch c.Mode {
		case store.ModeAgent:
			icon = "bot"
			if ag := a.agentByID(c.AgentID); ag != nil {
				icon, color = ag.Icon, ag.Color
			}
		case store.ModeResearch:
			icon, color = "telescope", "#38bdf8"
		}
		var trail widgets.El
		switch {
		case running:
			trail = giode.Spinner(widgets.St{Width: 14, Height: 14, Color: theme.T.Primary})
		case active || st.Hovered:
			trail = widgets.IconButton("side/conv/more/"+c.ID, "ellipsis", "Options", 16, theme.T.Muted, func() { a.openConversationMenu(c) })
		case c.Pinned:
			trail = widgets.Icon("pin", 13, theme.T.Faint)
		default:
			trail = giode.Box(widgets.St{})
		}
		fg := theme.T.Text2
		if active {
			fg = theme.T.Text
		}
		return widgets.Row(widgets.St{Gap: 10, Padding: giode.Inset{Left: 12, Right: 4, Top: 4, Bottom: 4}, MinHeight: 38, BorderRadius: 10, Background: bg, Width: -1},
			widgets.Icon(icon, 15, color),
			giode.Box(widgets.St{FlexGrow: 1}, widgets.Line(c.Title, theme.SizeCallout, fg)),
			trail)
	}, func() { a.openConversation(c) }, func() { a.openConversationMenu(c) })
	if !isMobile() {
		return row
	}
	return s.swipeRow(c, row)
}

// swipeRow lets a conversation row be swiped right to pin and left to
// delete.
func (s *sidebarView) swipeRow(c *store.Conversation, row widgets.El) widgets.El {
	a := s.a
	sw := giode.Keyed("swipe", "conv/"+c.ID, func() *swipeable.Swipeable {
		return giode.Swipeable(func() widgets.El { return s.current[c.ID] }, nil, nil)
	})
	pin := "Pin"
	if c.Pinned {
		pin = "Unpin"
	}
	// Refreshed every frame so labels and colors follow the pin state
	// and the theme.
	sw.SetActions(
		[]giode.SwipeAction{{Label: pin, Icon: widgets.Icon("pin", 18, "#ffffff"), Background: theme.T.Primary, Color: "#ffffff", OnTap: func() { a.togglePin(c) }}},
		[]giode.SwipeAction{{Label: "Delete", Icon: widgets.Icon("trash-2", 18, "#ffffff"), Background: theme.T.Danger, Color: "#ffffff", OnTap: func() {
			a.ask("Delete conversation?", "“"+c.Title+"” and its files will be removed from this device.", "Delete", true, func() {
				a.deleteConversation(c)
			})
		}}})
	if s.current == nil {
		s.current = map[string]widgets.El{}
	}
	s.current[c.ID] = row
	return sw
}
