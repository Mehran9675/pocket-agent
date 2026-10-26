package widgets

import (
	"strings"

	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// Avatar is a letter monogram in a colored circle.
func Avatar(letter, color string, size int) El {
	return giode.Box(St{
		Width: pr.Width(size), Height: pr.Height(size), BorderRadius: pr.BorderRadius(size / 2),
		Background: theme.Tint(color, 0.18), Align: giode.AlignCenter, Justify: giode.JustifyCenter,
	}, Text(strings.ToUpper(letter), float32(size)*0.42, color, giode.FontWeightBold))
}
