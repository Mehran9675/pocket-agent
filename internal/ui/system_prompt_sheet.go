package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) openSystemPrompt(c *store.Conversation) {
	key := "sys/" + c.ID
	in := a.input(key, "e.g. You are a meticulous editor. Reply in British English.", c.System, func(i *input.Input) { i.Multiline(true) })
	sh := a.openSheet("sysprompt", "Instructions for this chat", func() widgets.El {
		return widgets.Col(widgets.St{Gap: 10, Width: -1},
			widgets.Text("Replaces the default system prompt in this conversation only. Your global custom instructions and memory still apply.", theme.SizeFootnote, theme.T.Muted),
			giode.Box(widgets.St{Width: -1, MinHeight: 140}, in))
	})
	sh.footer = func() widgets.El {
		return widgets.Row(widgets.St{Gap: 10, Width: -1},
			widgets.Button("sys/clear", "Reset", "", widgets.BtnGhost, func() { in.SetText("") }),
			widgets.Spacer(),
			widgets.Button("sys/save", "Save", "", widgets.BtnPrimary, func() {
				c.System = strings.TrimSpace(in.Text())
				a.st.SaveConversation(c)
				a.closeSheet()
				a.toast(giode.ToastSuccess, "Instructions saved")
			}))
	}
}
