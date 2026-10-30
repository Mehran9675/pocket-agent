package ui

import (
	"strings"

	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) openRename(c *store.Conversation) {
	key := "rename/" + c.ID
	in := a.input(key, tr("Title"), c.Title)
	in.SetText(c.Title)
	save := func() {
		if t := strings.TrimSpace(in.Text()); t != "" {
			c.Title = t
			c.Version++
			a.st.SaveConversation(c)
		}
		a.closeSheet()
	}
	in.Submit(func(string) { save() })
	sh := a.openSheet("rename", tr("Rename conversation"), func() widgets.El {
		return widgets.Col(widgets.St{Gap: 14, Width: -1}, in)
	})
	sh.footer = func() widgets.El {
		return widgets.Row(widgets.St{Gap: 10, Width: -1}, widgets.Spacer(),
			widgets.Button("rename/cancel", tr("Cancel"), "", widgets.BtnSecondary, a.closeSheet),
			widgets.Button("rename/save", tr("Save"), "", widgets.BtnPrimary, save))
	}
	in.Focus()
}
