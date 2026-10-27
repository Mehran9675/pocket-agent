package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// pageScaffold is a titled page with a back button and scrolling body.
func (a *App) pageScaffold(key, title string, compact bool, actions widgets.El, body func() widgets.El) widgets.El {
	header := widgets.Row(widgets.St{Height: 56, Padding: giode.SymmetricInset(8, 0), Gap: 4, Width: -1},
		widgets.IconButton("back/"+key, "chevron-left", "Back", 22, theme.T.Text, a.back),
		widgets.Line(title, theme.SizeHeadline+1, theme.T.Text, giode.FontWeightSemiBold),
		widgets.Spacer(),
		widgets.When(actions != nil, func() widgets.El { return actions }),
	)
	sc := a.scroll("page/"+key, func() widgets.El {
		return giode.Box(widgets.St{Width: -1, Padding: giode.Inset{Left: 16, Right: 16, Top: 4, Bottom: 40}},
			widgets.Centered(720, giode.Box(widgets.St{Width: -1, FlexDirection: giode.FlexDirectionColumn, Gap: 18}, body())))
	})
	return widgets.Col(widgets.St{Width: -1, Height: -1},
		header,
		giode.Box(widgets.St{FlexGrow: 1, Width: -1}, sc),
	)
}
