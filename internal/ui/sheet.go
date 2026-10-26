package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/elements"
	pr "github.com/mehran9675/giode/styles/properties"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// sheet is a modal panel: a bottom sheet on phones, a centered card on
// wider screens.
type sheet struct {
	key   string
	title string
	body  func() widgets.El
	// footer stays pinned below the scrolling body.
	footer func() widgets.El
	width  int
}

func (a *App) openSheet(key, title string, body func() widgets.El) *sheet {
	a.sheet = &sheet{key: key, title: title, body: body, width: 520}
	a.sheetScrollReset(key)
	return a.sheet
}

func (a *App) sheetScrollReset(key string) {
	delete(a.scrolls, "sheet/"+key)
}

func (a *App) closeSheet() { a.sheet = nil }

func (a *App) sheetView(s *sheet) widgets.El {
	giode.HandleBack(a.closeSheet)
	win := dpSize(elements.WindowSize())
	compact := win.X < 700
	header := widgets.Row(widgets.St{Width: -1, Padding: giode.Inset{Left: 20, Right: 10, Top: 4, Bottom: 6}},
		widgets.Line(s.title, theme.SizeHeadline+1, theme.T.Text, giode.FontWeightBold),
		widgets.Spacer(),
		widgets.IconButton("sheet/close", "x", "Close", 20, theme.T.Muted, a.closeSheet),
	)
	maxH := win.Y * 85 / 100
	if !compact {
		maxH = win.Y * 80 / 100
	}
	body := a.fitScroll("sheet/"+s.key, func() widgets.El {
		return giode.Box(widgets.St{Width: -1, Padding: giode.Inset{Left: 16, Right: 16, Bottom: 16}, FlexDirection: giode.FlexDirectionColumn}, s.body())
	})
	kids := []widgets.El{}
	if compact {
		kids = append(kids, giode.Box(widgets.St{Width: -1, Justify: giode.JustifyCenter, Padding: giode.Inset{Top: 8, Bottom: 4}},
			giode.Box(widgets.St{Width: 38, Height: 5, BorderRadius: 3, Background: theme.T.BorderStrong})))
	} else {
		kids = append(kids, widgets.Gap(0, 8))
	}
	kids = append(kids, header, body)
	if s.footer != nil {
		kids = append(kids, widgets.HDivider(), giode.Box(widgets.St{Width: -1, Padding: giode.SymmetricInset(16, 12)}, s.footer()))
	}
	panelSt := widgets.St{
		FlexDirection: giode.FlexDirectionColumn, Background: theme.T.Elevated, MaxHeight: pr.MaxHeight(maxH),
		BoxShadow: giode.BoxShadow{Y: 8, Blur: 32, Color: theme.T.Shadow}, Overflow: giode.OverflowHidden,
	}
	if compact {
		panelSt.Width = -1
		panelSt.BorderRadii = giode.TopCorners(22)
		panelSt.Padding = giode.Inset{Bottom: 12}
	} else {
		panelSt.Width = pr.Width(s.width)
		panelSt.BorderRadius = 20
		panelSt.BorderWidth, panelSt.BorderColor = 1, theme.T.Border
	}
	panel := giode.Box(panelSt, kids...)
	layoutSt := widgets.St{Width: -1, Height: -1, FlexDirection: giode.FlexDirectionColumn, Align: giode.AlignCenter}
	if compact {
		layoutSt.Justify = giode.JustifyEnd
	} else {
		layoutSt.Justify = giode.JustifyCenter
	}
	return giode.Box(widgets.St{Width: -1, Height: -1},
		giode.Box(widgets.St{Position: giode.PositionAbsolute, Top: 0, Left: 0, Width: -1, Height: -1}, a.scrim("sheet/scrim", a.closeSheet)),
		giode.Box(widgets.St{Position: giode.PositionAbsolute, Top: 0, Left: 0, Width: -1, Height: -1, ZIndex: 1}, giode.Box(layoutSt, panel)),
	)
}
