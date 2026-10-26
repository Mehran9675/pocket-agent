package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) planCard(plan []tools.PlanItem, live bool) widgets.El {
	done := 0
	for _, p := range plan {
		if p.Status == "completed" {
			done++
		}
	}
	var rows []widgets.El
	for _, p := range plan {
		var icon widgets.El
		color := theme.T.Text2
		switch p.Status {
		case "completed":
			icon = widgets.Icon("circle-check", 17, theme.T.Success)
			color = theme.T.Muted
		case "in_progress":
			if live {
				icon = giode.Spinner(widgets.St{Width: 15, Height: 15, Color: theme.T.Primary})
			} else {
				icon = widgets.Icon("circle-dashed", 17, theme.T.Primary)
			}
			color = theme.T.Text
		default:
			icon = widgets.Icon("circle", 17, theme.T.Faint)
		}
		st := widgets.TextStyle(theme.SizeFootnote, color)
		if p.Status == "completed" {
			st.TextDecoration = giode.TextDecorationLineThrough
		}
		rows = append(rows, widgets.Row(widgets.St{Gap: 10, Align: giode.AlignStart, Width: -1},
			giode.Box(widgets.St{Padding: giode.Inset{Top: 1}}, icon), giode.Box(widgets.St{FlexGrow: 1}, giode.Text(p.Step, st))))
	}
	progress := float32(0)
	if len(plan) > 0 {
		progress = float32(done) / float32(len(plan))
	}
	return widgets.Card(widgets.St{Gap: 12},
		widgets.Row(widgets.St{Gap: 8, Width: -1},
			widgets.Icon("list-todo", 16, theme.T.Primary),
			widgets.Text("Plan", theme.SizeCallout, theme.T.Text, giode.FontWeightBold),
			widgets.Spacer(),
			widgets.Text(fmt.Sprintf("%d of %d", done, len(plan)), theme.SizeFootnote, theme.T.Muted)),
		giode.Progress(progress, widgets.St{Color: theme.T.Primary, Background: theme.T.Surface2, Height: 4}),
		widgets.Col(widgets.St{Gap: 8, Width: -1}, rows...),
	)
}
