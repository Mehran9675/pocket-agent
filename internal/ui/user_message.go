package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) userMessage(c *store.Conversation, m *store.Message) widgets.El {
	var kids []widgets.El
	if len(m.Attachments) > 0 {
		var atts []widgets.El
		for _, at := range m.Attachments {
			atts = append(atts, a.attachmentChip("msgatt/"+at.ID, at, false, nil))
		}
		kids = append(kids, giode.Row(widgets.St{Gap: 8, Wrap: giode.WrapWrap, Justify: giode.JustifyEnd}, atts...))
	}
	if strings.TrimSpace(m.Text) != "" {
		kids = append(kids, giode.Box(widgets.St{
			Padding: giode.SymmetricInset(16, 11), BorderRadius: 20, Background: theme.T.UserBubble,
			BorderRadii: giode.Corners{TopLeft: 20, TopRight: 20, BottomLeft: 20, BottomRight: 6},
		}, giode.Text(m.Text, widgets.St{FontSize: theme.SizeBody, Color: theme.T.Text, LineHeight: 22})))
	}
	running := a.isRunning(c)
	actions := widgets.Row(widgets.St{Gap: 0},
		widgets.IconButton("u/copy/"+m.ID, "copy", "Copy", 15, theme.T.Faint, func() { a.copyText(m.Text) }),
		widgets.When(!running, func() widgets.El {
			return widgets.IconButton("u/edit/"+m.ID, "pencil", "Edit", 15, theme.T.Faint, func() { a.editMessage(c, m) })
		}),
	)
	kids = append(kids, actions)
	return giode.Box(widgets.St{Width: -1, Justify: giode.JustifyEnd, Padding: giode.Inset{Left: 48}},
		widgets.Col(widgets.St{Gap: 4, Align: giode.AlignEnd}, kids...))
}
