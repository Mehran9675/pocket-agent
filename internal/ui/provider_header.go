package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func providerHeader(icon, color, name, status string) widgets.El {
	return widgets.Row(widgets.St{Gap: 8, Width: -1, Padding: giode.Inset{Left: 4, Top: 12, Bottom: 4}},
		widgets.Icon(icon, 14, color),
		widgets.Text(strings.ToUpper(name), theme.SizeMicro, theme.T.Muted, giode.FontWeightSemiBold),
		widgets.Spacer(),
		widgets.Text(status, theme.SizeCaption, theme.T.Faint))
}
