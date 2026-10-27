package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// inspectorRelevant reports whether the current chat has agent or
// research detail worth a side panel.
func (a *App) inspectorRelevant() bool {
	c := a.current
	return c != nil && (c.Mode == store.ModeAgent || c.Mode == store.ModeResearch) && len(c.Messages) > 0
}

// inspector is the right-hand panel on wide screens: the live plan,
// workspace files and sources of the current conversation.
func (a *App) inspector() widgets.El {
	c := a.current
	var plan []tools.PlanItem
	var sources []tools.Source
	seen := map[string]bool{}
	running := a.isRunning(c)
	for _, m := range c.Messages {
		if len(m.Plan) > 0 {
			plan = m.Plan
		}
		for _, s := range m.Sources() {
			if !seen[s.URL] {
				seen[s.URL] = true
				sources = append(sources, s)
			}
		}
	}
	files := a.workspaceFiles(c)
	body := func() widgets.El {
		var kids []widgets.El
		title := "Details"
		if c.Mode == store.ModeAgent {
			if ag := a.agentByID(c.AgentID); ag != nil {
				title = ag.Name
				kids = append(kids, widgets.Row(widgets.St{Gap: 12, Width: -1}, widgets.IconTile(ag.Icon, ag.Color, 40),
					widgets.Col(widgets.St{Gap: 2, FlexGrow: 1}, widgets.Text(ag.Name, theme.SizeHeadline, theme.T.Text, giode.FontWeightSemiBold), widgets.Text(ag.Description, theme.SizeFootnote, theme.T.Muted))))
			}
		} else {
			title = "Deep research"
			kids = append(kids, widgets.Row(widgets.St{Gap: 12, Width: -1}, widgets.IconTile("telescope", "#38bdf8", 40),
				widgets.Col(widgets.St{Gap: 2, FlexGrow: 1}, widgets.Text(title, theme.SizeHeadline, theme.T.Text, giode.FontWeightSemiBold), widgets.Text("Sources gathered across this conversation", theme.SizeFootnote, theme.T.Muted))))
		}
		status := "Idle"
		if running {
			status = "Working…"
		}
		kids = append(kids, widgets.Row(widgets.St{Gap: 8}, widgets.Dot(map[bool]string{true: theme.T.Success, false: theme.T.Faint}[running], 8), widgets.Text(status, theme.SizeFootnote, theme.T.Text2)))
		if len(plan) > 0 {
			kids = append(kids, a.planCard(plan, running))
		}
		if c.Mode == store.ModeAgent {
			var rows []widgets.El
			for _, f := range files {
				rows = append(rows, a.fileRow(c, f))
			}
			if len(rows) == 0 {
				rows = append(rows, giode.Box(widgets.St{Padding: giode.UniformInset(8)}, widgets.Text("Files the agent writes appear here.", theme.SizeFootnote, theme.T.Muted)))
			}
			kids = append(kids, widgets.SectionLabel("Workspace"), widgets.Card(widgets.St{Padding: giode.UniformInset(6), Gap: 0}, rows...))
		}
		if len(sources) > 0 {
			var rows []widgets.El
			for i, s := range sources {
				s := s
				rows = append(rows, widgets.Tap(fmt.Sprintf("insp/src/%d", i), s.Title, func(st giode.PressState) widgets.El {
					return widgets.Col(widgets.St{Gap: 2, Padding: giode.SymmetricInset(10, 8), BorderRadius: 10, Background: widgets.HoverBg(st, "transparent"), Width: -1},
						widgets.Line(fmt.Sprintf("%d. %s", i+1, s.Title), theme.SizeFootnote, theme.T.Text, giode.FontWeightMedium),
						widgets.Line(s.URL, theme.SizeCaption, theme.T.Faint))
				}, func() { a.openLink(s.URL) }))
			}
			kids = append(kids, widgets.SectionLabel(fmt.Sprintf("Sources · %d", len(sources))), widgets.Card(widgets.St{Padding: giode.UniformInset(4), Gap: 0}, rows...))
		}
		return widgets.Col(widgets.St{Gap: 16, Width: -1, Padding: giode.UniformInset(16)}, kids...)
	}
	return widgets.Col(widgets.St{Width: -1, Height: -1},
		widgets.Row(widgets.St{Height: 56, Padding: giode.SymmetricInset(16, 0), Width: -1}, widgets.Text("Inspector", theme.SizeCallout, theme.T.Muted, giode.FontWeightSemiBold)),
		giode.Box(widgets.St{FlexGrow: 1, Width: -1}, a.scroll("inspector", body)))
}
