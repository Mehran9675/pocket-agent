package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// ListRow is a settings-style row: icon tile, title, subtitle, trailing.
func ListRow(key, icon, iconColor, title, subtitle string, trailing El, onClick func()) El {
	build := func(s giode.PressState) El {
		bg := "transparent"
		if onClick != nil {
			bg = HoverBg(s, "transparent")
		}
		texts := []El{Line(title, theme.SizeBody, theme.T.Text, giode.FontWeightMedium)}
		if subtitle != "" {
			texts = append(texts, Text(subtitle, theme.SizeFootnote, theme.T.Muted))
		}
		kids := []El{}
		if icon != "" {
			kids = append(kids, IconTile(icon, iconColor, 30))
		}
		kids = append(kids, Col(St{Gap: 2, FlexGrow: 1}, texts...))
		if trailing != nil {
			kids = append(kids, trailing)
		} else if onClick != nil {
			kids = append(kids, Icon("chevron-right", 18, theme.T.Faint))
		}
		return Row(St{Gap: 12, Padding: giode.SymmetricInset(12, 11), Background: bg, Width: -1}, kids...)
	}
	if onClick == nil {
		return build(giode.PressState{})
	}
	return Tap(key, title, build, onClick)
}
