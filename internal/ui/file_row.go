package ui

import (
	"path/filepath"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) fileRow(c *store.Conversation, f string) widgets.El {
	full := filepath.Join(a.st.Workspace(c.ID), filepath.FromSlash(f))
	return widgets.Tap("file/"+c.ID+"/"+f, f, func(s giode.PressState) widgets.El {
		return widgets.Row(widgets.St{Gap: 10, Padding: giode.SymmetricInset(8, 8), BorderRadius: 10, Background: widgets.HoverBg(s, "transparent"), Width: -1},
			widgets.IconTile(fileIcon(f), theme.T.Primary, 28),
			giode.Box(widgets.St{FlexGrow: 1}, widgets.Line(f, theme.SizeFootnote, theme.T.Text, giode.FontWeightMedium)),
			widgets.IconButton("file/share/"+c.ID+"/"+f, "share", tr("Share file"), 15, theme.T.Muted, func() {
				a.shareFile(full, mimeOf(f))
			}),
			widgets.Icon("chevron-right", 15, theme.T.Faint))
	}, func() { a.navigate("file/" + c.ID + "/" + f) })
}
