package widgets

import (
	"github.com/mehran9675/giode"
)

// IconButton is a borderless icon button with a hover/press plate.
func IconButton(key, icon, label string, size int, color string, onClick func()) El {
	return Tap(key, label, func(s giode.PressState) El {
		pad := (size*2 - size) / 2
		if pad < 6 {
			pad = 6
		}
		return giode.Box(St{
			Padding:      giode.UniformInset(pad),
			BorderRadius: 10,
			Background:   HoverBg(s, "transparent"),
			Align:        giode.AlignCenter, Justify: giode.JustifyCenter,
		}, Icon(icon, size, color))
	}, onClick)
}
