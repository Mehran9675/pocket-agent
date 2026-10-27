package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func stat(label, value string) widgets.El {
	return widgets.Col(widgets.St{Gap: 2}, widgets.Text(label, theme.SizeCaption, theme.T.Muted), widgets.Text(value, theme.SizeCallout, theme.T.Text, giode.FontWeightSemiBold))
}
