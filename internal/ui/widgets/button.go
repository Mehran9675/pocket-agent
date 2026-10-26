package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// ButtonKind styles buttons.
type ButtonKind int

const (
	BtnPrimary ButtonKind = iota
	BtnSecondary
	BtnGhost
	BtnDanger
)

// Button is a text button with an optional leading icon.
func Button(key, label, icon string, kind ButtonKind, onClick func(), opts ...St) El {
	var o St
	if len(opts) > 0 {
		o = opts[0]
	}
	return Tap(key, label, func(s giode.PressState) El {
		bg, fg, border := theme.T.Primary, theme.T.OnPrimary, ""
		switch kind {
		case BtnSecondary:
			bg, fg, border = theme.T.Surface2, theme.T.Text, theme.T.Border
		case BtnGhost:
			bg, fg = "transparent", theme.T.Text2
		case BtnDanger:
			bg, fg = theme.T.DangerSoft, theme.T.Danger
		}
		if s.Disabled {
			bg, fg = theme.T.Surface2, theme.T.Faint
		}
		st := St{
			Padding: giode.SymmetricInset(16, 10), BorderRadius: 12,
			Background: bg, Align: giode.AlignCenter, Justify: giode.JustifyCenter, Gap: 8,
		}
		if o.Width != 0 || o.FlexGrow != 0 {
			st.Width = -1
		}
		if border != "" {
			st.BorderWidth, st.BorderColor = 1, border
		}
		if s.Pressed && !s.Disabled {
			st.Opacity = 0.82
		} else if s.Hovered && !s.Disabled {
			st.Opacity = 0.92
		}
		kids := []El{}
		if icon != "" {
			kids = append(kids, Icon(icon, 17, fg))
		}
		kids = append(kids, Text(label, theme.SizeCallout, fg, giode.FontWeightMedium))
		return giode.Row(st, kids...)
	}, onClick, St{Width: o.Width, FlexGrow: o.FlexGrow})
}
