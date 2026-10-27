package widgets

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/scroll"
)

// VRow is one row of a VirtualColumn.
type VRow struct {
	// Key identifies the row across frames; Token changes whenever its
	// content (and so its height) may have.
	Key   string
	Token string
	// Gap is the space above the row, in dp.
	Gap   int
	Build func() El
}

// VirtualColumn is a column of rows of any height inside a Scroll that
// lays out only the rows in or near the viewport, like React Native's
// FlatList: every other row is skipped using the height it had when it
// was last laid out. A row is laid out at least once, when it first
// appears or its Token changes, so heights are always measured, never
// guessed, and the content never jumps.
//
// Each laid-out row is memoized under its key, so rows in view replay
// their drawing unless they changed.
type VirtualColumn struct {
	scroll  *scroll.Scroll
	rows    []VRow
	heights map[string]rowHeight
	width   int
	// top is the column's offset in the scroll content, from the last
	// frame, to compare rows against the viewport.
	top int
}

type rowHeight struct {
	token string
	px    int
	seen  bool
}

// NewVirtualColumn returns a column for the content of s.
func NewVirtualColumn(s *scroll.Scroll) *VirtualColumn {
	return &VirtualColumn{scroll: s, heights: map[string]rowHeight{}}
}

// SetRows replaces the rows, for this frame.
func (v *VirtualColumn) SetRows(rows []VRow) { v.rows = rows }

// SetTop tells the column how far below the top of the scroll content
// it starts, in px.
func (v *VirtualColumn) SetTop(px int) { v.top = px }

// Layout implements giode's Element.
func (v *VirtualColumn) Layout(gtx layout.Context) layout.Dimensions {
	width := gtx.Constraints.Max.X
	if width != v.width {
		v.width = width
		clear(v.heights)
	}
	view := v.scroll.ViewportHeight()
	from := int(v.scroll.Offset()) - v.top
	// One screen of rows beyond each edge stays laid out, so a fling
	// never shows a gap.
	lo, hi := from-view, from+2*view

	rgtx := gtx
	rgtx.Constraints.Min = image.Pt(width, 0)
	y := 0
	for i, r := range v.rows {
		if i > 0 {
			y += gtx.Dp(unit.Dp(r.Gap))
		}
		h, known := v.heights[r.Key]
		known = known && h.token == r.Token
		if known && view > 0 && (y+h.px < lo || y > hi) {
			h.seen = true
			v.heights[r.Key] = h
			y += h.px
			continue
		}
		off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
		dims := giode.Memo("vrow/"+r.Key, r.Build, r.Token).Layout(rgtx)
		off.Pop()
		v.heights[r.Key] = rowHeight{token: r.Token, px: dims.Size.Y, seen: true}
		y += dims.Size.Y
	}
	// Forget rows that are gone.
	for k, h := range v.heights {
		if !h.seen {
			delete(v.heights, k)
			continue
		}
		h.seen = false
		v.heights[k] = h
	}
	return layout.Dimensions{Size: image.Pt(width, y)}
}
