package ui

import (
	"time"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) toolRow(c *store.Conversation, m *store.Message, t *store.ToolRun) widgets.El {
	key := m.ID + "/tool/" + t.CallID
	open := a.expanded[key]
	icon, title := a.toolMeta(t)
	if t.Status == store.ToolPending {
		return a.approvalCard(c, t, icon, title)
	}
	header := widgets.Tap(key, title, func(s giode.PressState) widgets.El {
		var status widgets.El
		switch t.Status {
		case store.ToolRunning:
			status = giode.Spinner(widgets.St{Width: 14, Height: 14, Color: theme.T.Primary})
		case store.ToolError:
			status = widgets.Icon("circle-x", 15, theme.T.Danger)
		case store.ToolDenied:
			status = widgets.Icon("ban", 15, theme.T.Warning)
		case store.ToolCanceled:
			status = widgets.Icon("circle-x", 15, theme.T.Faint)
		default:
			status = widgets.Icon("check", 15, theme.T.Success)
		}
		detail := ""
		if t.Status == store.ToolDenied {
			detail = "Declined"
		} else if t.Duration >= 100*time.Millisecond {
			detail = fmtDuration(t.Duration)
		}
		chev := "chevron-right"
		if open {
			chev = "chevron-down"
		}
		return widgets.Row(widgets.St{Gap: 10, Padding: giode.SymmetricInset(8, 6), BorderRadius: 10, Background: widgets.HoverBg(s, "transparent"), Width: -1},
			giode.Box(widgets.St{Width: 26, Height: 26, BorderRadius: 8, Background: theme.T.Surface2, Align: giode.AlignCenter, Justify: giode.JustifyCenter},
				widgets.Icon(icon, 14, theme.T.Text2)),
			giode.Box(widgets.St{FlexGrow: 1}, widgets.Line(title, theme.SizeFootnote, theme.T.Text2, giode.FontWeightMedium)),
			widgets.Text(detail, theme.SizeCaption, theme.T.Faint),
			status,
			widgets.Icon(chev, 14, theme.T.Faint),
		)
	}, func() { a.expanded[key] = !open })
	if !open {
		return header
	}
	args := prettyJSON(t.Args)
	result := t.Result
	if r := []rune(result); len(r) > 3000 {
		result = string(r[:3000]) + "\n…"
	}
	detail := widgets.Col(widgets.St{Gap: 8, Width: -1, Padding: giode.Inset{Left: 44, Right: 8, Bottom: 6}},
		widgets.MonoBox("Input", args),
		widgets.When(result != "", func() widgets.El { return widgets.MonoBox("Output", result) }),
	)
	return widgets.Col(widgets.St{Gap: 4, Width: -1}, header, detail)
}
