package widgets

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/elements"
)

// Centered lays child out at the full available width up to maxWidth
// dp, centered horizontally - a readable content column. (Giode's
// MaxWidth doesn't cap a Width: -1 child yet.)
func Centered(maxWidth int, child El) El {
	return giode.Raw(func(gtx layout.Context) layout.Dimensions {
		avail := gtx.Constraints.Max.X
		w := min(avail, gtx.Dp(unit.Dp(maxWidth)))
		cgtx := gtx
		cgtx.Constraints.Min.X, cgtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		t := elements.TrackChild()
		dims := child.Layout(cgtx)
		t.End()
		call := macro.Stop()
		off := image.Pt((avail-w)/2, 0)
		t.Place(off)
		st := op.Offset(off).Push(gtx.Ops)
		call.Add(gtx.Ops)
		st.Pop()
		return layout.Dimensions{Size: image.Pt(avail, dims.Size.Y), Baseline: dims.Baseline}
	})
}
