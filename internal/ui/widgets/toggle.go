package widgets

import (
	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// Toggle is an iOS-style switch.
func Toggle(key string, on bool, onChange func(bool)) El {
	return Tap(key, "Toggle", func(s giode.PressState) El {
		track := theme.T.BorderStrong
		if on {
			track = theme.T.Primary
		}
		left := 3
		if on {
			left = 21
		}
		knob := giode.Box(St{
			Position: giode.PositionAbsolute, Left: pr.Left(left), Top: 3,
			Width: 20, Height: 20, BorderRadius: 10, Background: "#ffffff",
			BoxShadow: giode.BoxShadow{Y: 1, Blur: 3, Color: "#00000040"},
		})
		return giode.Box(St{Width: 44, Height: 26, BorderRadius: 13, Background: track}, knob)
	}, func() { onChange(!on) })
}
