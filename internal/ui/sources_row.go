package ui

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) sourcesRow(id string, srcs []tools.Source) widgets.El {
	key := id + "/sources"
	open := a.expanded[key]
	shown := srcs
	if !open && len(shown) > 6 {
		shown = shown[:6]
	}
	var chips []widgets.El
	for i, s := range shown {
		s := s
		host := s.URL
		if u, err := url.Parse(s.URL); err == nil {
			host = strings.TrimPrefix(u.Host, "www.")
		}
		title := s.Title
		if title == "" {
			title = host
		}
		chips = append(chips, widgets.Tap(fmt.Sprintf("%s/%d", key, i), title, func(st giode.PressState) widgets.El {
			letter := "•"
			if host != "" {
				letter = strings.ToUpper(host[:1])
			}
			return widgets.Row(widgets.St{Gap: 8, Padding: giode.Inset{Left: 6, Right: 10, Top: 5, Bottom: 5}, BorderRadius: 10, MaxWidth: 260,
				Background: widgets.HoverBg(st, theme.T.Surface), BorderWidth: 1, BorderColor: theme.T.Border},
				giode.Box(widgets.St{Width: 20, Height: 20, BorderRadius: 6, Background: theme.T.Surface2, Align: giode.AlignCenter, Justify: giode.JustifyCenter},
					widgets.Text(letter, theme.SizeMicro, theme.T.Text2, giode.FontWeightSemiBold)),
				widgets.Col(widgets.St{Gap: 0}, widgets.Line(fmt.Sprintf("%d. %s", i+1, title), theme.SizeCaption, theme.T.Text2, giode.FontWeightMedium), widgets.Line(host, theme.SizeMicro, theme.T.Faint)))
		}, func() { a.openLink(s.URL) }))
	}
	if len(srcs) > 6 {
		label := fmt.Sprintf("+%d more", len(srcs)-6)
		if open {
			label = "Show less"
		}
		chips = append(chips, widgets.Chip(key+"/more", label, "", false, func() { a.expanded[key] = !open }))
	}
	return widgets.Col(widgets.St{Gap: 8, Width: -1},
		widgets.Row(widgets.St{Gap: 6}, widgets.Icon("book-open", 14, theme.T.Muted), widgets.Text(fmt.Sprintf("%d sources", len(srcs)), theme.SizeFootnote, theme.T.Muted, giode.FontWeightMedium)),
		giode.Row(widgets.St{Gap: 6, Wrap: giode.WrapWrap, Width: -1}, chips...),
	)
}
