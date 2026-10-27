package widgets

import (
	"gioui.org/unit"

	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// CodeText renders selectable monospace text with 4-column tab stops.
// key identifies it for selection.
func CodeText(key any, s string, size float32, color string) El {
	return giode.Text(s, St{
		FontFamily: theme.FontMono, FontSize: unit.Sp(size), Color: color,
		LineHeight: pr.LineHeight(size * 1.55), TabWidth: 4,
	}).Selectable(key)
}
