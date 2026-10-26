package widgets

import (
	"strings"

	"gioui.org/unit"

	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// CodeText renders monospace text. Tabs become four spaces: the
// shaper has no tab stops and would draw a missing-glyph box.
func CodeText(s string, size float32, color string) El {
	return giode.Text(strings.ReplaceAll(s, "\t", "    "), St{
		FontFamily: theme.FontMono, FontSize: unit.Sp(size), Color: color,
		LineHeight: pr.LineHeight(size * 1.55),
	})
}
