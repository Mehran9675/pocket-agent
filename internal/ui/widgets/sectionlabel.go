package widgets

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// SectionLabel is a small caps heading above grouped content.
func SectionLabel(s string) El {
	return giode.Box(St{Padding: giode.Inset{Left: 4, Bottom: 6, Top: 4}},
		giode.Text(strings.ToUpper(s), St{FontSize: theme.SizeMicro, Color: theme.T.Muted, FontWeight: giode.FontWeightBold}))
}
