package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) approvalCard(c *store.Conversation, t *store.ToolRun, icon, title string) widgets.El {
	args := prettyJSON(t.Args)
	if r := []rune(args); len(r) > 1200 {
		args = string(r[:1200]) + "\n…"
	}
	return widgets.Col(widgets.St{
		Gap: 12, Width: -1, Padding: giode.UniformInset(14), BorderRadius: 16,
		Background: theme.T.WarningSoft, BorderWidth: 1, BorderColor: theme.Tint(theme.T.Warning, 0.35),
	},
		widgets.Row(widgets.St{Gap: 10},
			widgets.IconTile("shield-check", theme.T.Warning, 30),
			widgets.Col(widgets.St{Gap: 2, FlexGrow: 1},
				widgets.Text("Allow this action?", theme.SizeCallout, theme.T.Text, giode.FontWeightBold),
				widgets.Line(title, theme.SizeFootnote, theme.T.Text2)),
		),
		widgets.MonoBox("Details", args),
		giode.Row(widgets.St{Gap: 8, Wrap: giode.WrapWrap},
			widgets.Button("ap/allow/"+t.CallID, "Allow once", "check", widgets.BtnPrimary, func() { a.answerApproval(c, t.CallID, t.Name, true, false) }),
			widgets.Button("ap/always/"+t.CallID, "Always allow", "", widgets.BtnSecondary, func() { a.answerApproval(c, t.CallID, t.Name, true, true) }),
			widgets.Button("ap/deny/"+t.CallID, "Deny", "x", widgets.BtnGhost, func() { a.answerApproval(c, t.CallID, t.Name, false, false) }),
		),
	)
}
