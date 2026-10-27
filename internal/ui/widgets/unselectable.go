package widgets

import (
	"gioui.org/layout"
	"github.com/mehran9675/giode/components/kit"
)

// Unselectable keeps the text in el out of an enclosing SelectionArea:
// labels, statuses and other interface text that isn't content.
func Unselectable(el El) El { return unselectable{el} }

type unselectable struct{ el El }

func (u unselectable) Layout(gtx layout.Context) layout.Dimensions {
	return kit.Unselectable(u.el.Layout)(gtx)
}
