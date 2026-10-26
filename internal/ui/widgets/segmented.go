package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// Segmented is a Segmented control.
func Segmented(key string, options []string, icons []string, selected int, onChange func(int)) El {
	var kids []El
	for i, o := range options {
		i, o := i, o
		icon := ""
		if i < len(icons) {
			icon = icons[i]
		}
		kids = append(kids, Tap(key+"/"+o, o, func(s giode.PressState) El {
			bg, fg := "transparent", theme.T.Muted
			var shadow giode.BoxShadow
			if i == selected {
				bg, fg = theme.T.Elevated, theme.T.Text
				if theme.T.Dark {
					bg = theme.T.BorderStrong
				}
				shadow = giode.BoxShadow{Y: 1, Blur: 3, Color: theme.T.Shadow}
			} else if s.Hovered {
				fg = theme.T.Text2
			}
			inner := []El{}
			if icon != "" {
				inner = append(inner, Icon(icon, 15, fg))
			}
			inner = append(inner, Text(o, theme.SizeFootnote, fg, giode.FontWeightMedium))
			return Row(St{
				Gap: 6, Padding: giode.SymmetricInset(12, 7), BorderRadius: 9, Justify: giode.JustifyCenter,
				Background: bg, BoxShadow: shadow, FlexGrow: 1,
			}, inner...)
		}, func() { onChange(i) }))
	}
	return Row(St{Padding: giode.UniformInset(3), BorderRadius: 12, Background: theme.T.Surface2, Gap: 2}, kids...)
}
