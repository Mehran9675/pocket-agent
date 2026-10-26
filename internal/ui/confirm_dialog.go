package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// confirmState is a yes/no question.
type confirmState struct {
	title, message, action string
	danger                 bool
	onYes                  func()
}

func (a *App) ask(title, message, action string, danger bool, onYes func()) {
	a.confirm = &confirmState{title: title, message: message, action: action, danger: danger, onYes: onYes}
}

func (a *App) confirmView(c *confirmState) widgets.El {
	giode.HandleBack(func() { a.confirm = nil })
	kind := widgets.BtnPrimary
	if c.danger {
		kind = widgets.BtnDanger
	}
	panel := giode.Box(widgets.St{
		FlexDirection: giode.FlexDirectionColumn, Width: 380, Background: theme.T.Elevated, BorderRadius: 20,
		BorderWidth: 1, BorderColor: theme.T.Border, Padding: giode.UniformInset(22), Gap: 10,
		BoxShadow: giode.BoxShadow{Y: 8, Blur: 32, Color: theme.T.Shadow},
	},
		widgets.Text(c.title, theme.SizeHeadline+1, theme.T.Text, giode.FontWeightBold),
		widgets.Text(c.message, theme.SizeCallout, theme.T.Text2),
		widgets.Gap(0, 6),
		widgets.Row(widgets.St{Gap: 10, Justify: giode.JustifyEnd, Width: -1},
			widgets.Spacer(),
			widgets.Button("confirm/no", "Cancel", "", widgets.BtnSecondary, func() { a.confirm = nil }),
			widgets.Button("confirm/yes", c.action, "", kind, func() {
				a.confirm = nil
				c.onYes()
			}),
		),
	)
	return giode.Box(widgets.St{Width: -1, Height: -1},
		giode.Box(widgets.St{Position: giode.PositionAbsolute, Top: 0, Left: 0, Width: -1, Height: -1}, a.scrim("confirm/scrim", func() { a.confirm = nil })),
		giode.Box(widgets.St{Position: giode.PositionAbsolute, Top: 0, Left: 0, Width: -1, Height: -1, ZIndex: 1},
			giode.Box(widgets.St{Width: -1, Height: -1, Align: giode.AlignCenter, Justify: giode.JustifyCenter, Padding: giode.UniformInset(20), FlexDirection: giode.FlexDirectionColumn}, panel)),
	)
}
