package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) filesCard(c *store.Conversation, files []string) widgets.El {
	var rows []widgets.El
	for _, f := range files {
		f := f
		rows = append(rows, a.fileRow(c, f))
	}
	return widgets.Card(widgets.St{Gap: 4, Padding: giode.UniformInset(8)},
		giode.Box(widgets.St{Padding: giode.Inset{Left: 6, Top: 4, Bottom: 4}}, widgets.Row(widgets.St{Gap: 6}, widgets.Icon("folder", 14, theme.T.Muted), widgets.Text(tr("Workspace files"), theme.SizeFootnote, theme.T.Muted, giode.FontWeightMedium))),
		widgets.Col(widgets.St{Gap: 0, Width: -1}, rows...))
}
