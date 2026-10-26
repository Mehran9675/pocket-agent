package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) instructionsBody() widgets.El {
	in := a.input("instructions", "e.g. I'm a product designer in Berlin. Be concise; use metric units; show code in TypeScript.", a.settings.Instructions, func(i *input.Input) { i.Multiline(true) })
	return widgets.Col(widgets.St{Gap: 14, Width: -1},
		widgets.Text("Pocket Agent considers these in every conversation: who you are, how you like answers, what to avoid.", theme.SizeFootnote, theme.T.Muted),
		giode.Box(widgets.St{Width: -1, MinHeight: 180}, in),
		widgets.Row(widgets.St{Gap: 10, Width: -1}, widgets.Spacer(),
			widgets.Button("instr/save", "Save", "check", widgets.BtnPrimary, func() {
				a.settings.Instructions = strings.TrimSpace(in.Text())
				a.saveSettings()
				a.toast(giode.ToastSuccess, "Instructions saved")
			})),
	)
}
