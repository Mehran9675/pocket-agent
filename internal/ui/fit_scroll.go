package ui

import (
	"gioui.org/layout"
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/kit"
	"github.com/mehran9675/giode/components/scroll"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// fitScroll is a scroll container that is only as tall as its content
// (up to the space available), using the height measured last frame.
func (a *App) fitScroll(key string, content func() widgets.El) widgets.El {
	sc := a.scroll(key, content).(*scroll.Scroll)
	return giode.Raw(func(gtx layout.Context) layout.Dimensions {
		h := sc.ContentHeight()
		if h > 0 && h < gtx.Constraints.Max.Y {
			gtx.Constraints.Max.Y = h
		}
		dims := sc.Layout(gtx)
		if sc.ContentHeight() != h {
			kit.RequestFrame(gtx)
		}
		return dims
	})
}
