package widgets

import "gioui.org/layout"

// Measure lays out el and reports its height in dp, for layouts that
// make room for something drawn over them.
func Measure(el El, height func(dp float32)) El { return measure{el, height} }

type measure struct {
	el     El
	height func(dp float32)
}

func (m measure) Layout(gtx layout.Context) layout.Dimensions {
	dims := m.el.Layout(gtx)
	m.height(float32(dims.Size.Y) / gtx.Metric.PxPerDp)
	return dims
}
