package ui

import (
	"gioui.org/layout"
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/elements"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// overlays registers the open sheet and dialog above the view.
func (a *App) overlays() widgets.El {
	return giode.Raw(func(gtx layout.Context) layout.Dimensions {
		if s := a.sheet; s != nil {
			elements.RegisterOverlay(func() widgets.El { return a.sheetView(s) })
		}
		if c := a.confirm; c != nil {
			elements.RegisterOverlay(func() widgets.El { return a.confirmView(c) })
		}
		return layout.Dimensions{}
	})
}
