package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// Group wraps settings rows in a rounded container with dividers.
func Group(rows ...El) El {
	var kids []El
	for i, r := range rows {
		if i > 0 {
			kids = append(kids, giode.Box(St{Padding: giode.Inset{Left: 52}}, HDivider()))
		}
		kids = append(kids, r)
	}
	return giode.Box(St{
		FlexDirection: giode.FlexDirectionColumn, Background: theme.T.Surface, BorderRadius: 14,
		BorderWidth: 1, BorderColor: theme.T.Border, Overflow: giode.OverflowHidden,
	}, kids...)
}
