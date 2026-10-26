package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// Chip is a rounded toggle pill.
func Chip(key, label, icon string, active bool, onClick func()) El {
	return Tap(key, label, func(s giode.PressState) El {
		bg, fg, border := "transparent", theme.T.Text2, theme.T.Border
		if active {
			bg, fg, border = theme.T.PrimarySoft, theme.T.PrimaryText, theme.Tint(theme.T.Primary, 0.4)
		} else if s.Hovered || s.Pressed {
			bg = theme.T.Hover
		}
		kids := []El{}
		if icon != "" {
			kids = append(kids, Icon(icon, 15, fg))
		}
		if label != "" {
			kids = append(kids, Text(label, theme.SizeFootnote, fg, giode.FontWeightMedium))
		}
		return Row(St{
			Gap: 6, Padding: giode.SymmetricInset(11, 7), BorderRadius: 999,
			Background: bg, BorderWidth: 1, BorderColor: border,
		}, kids...)
	}, onClick)
}
