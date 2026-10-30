package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) agentsPage(compact bool) widgets.El {
	return a.pageScaffold("agents", tr("Agents"), compact,
		widgets.Button("agents/new", ifStr(compact, tr("New"), tr("New agent")), "plus", widgets.BtnPrimary, func() {
			ag := &store.Agent{
				ID: store.NewID(), Name: tr("New agent"), Icon: "bot", Color: agentColors[len(a.agents)%len(agentColors)],
				Description: tr("Describe what this agent is for."), MaxSteps: 20,
				Instructions: tr("You are a capable autonomous agent. Plan with update_plan, use your tools, and finish with a clear summary."),
				Tools:        []string{"update_plan", "web_search", "fetch_url", "run_javascript", "write_file", "read_file", "list_files"},
			}
			a.agents = append(a.agents, ag)
			a.st.SaveAgents(a.agents)
			a.navigate("agents/edit/" + ag.ID)
		}),
		a.agentsBody)
}

func (a *App) agentsBody() widgets.El {
	var cards []widgets.El
	for _, ag := range a.agents {
		ag := ag
		cards = append(cards, widgets.Tap("agent/"+ag.ID, agentName(ag), func(s giode.PressState) widgets.El {
			return widgets.Col(widgets.St{
				Gap: 10, Padding: giode.UniformInset(16), BorderRadius: 16, Width: -1,
				Background: widgets.HoverBg(s, theme.T.Surface), BorderWidth: 1, BorderColor: theme.T.Border,
			},
				widgets.Row(widgets.St{Gap: 12, Width: -1},
					widgets.IconTile(ag.Icon, ag.Color, 42),
					widgets.Col(widgets.St{Gap: 2, FlexGrow: 1},
						widgets.Row(widgets.St{Gap: 8}, widgets.Text(agentName(ag), theme.SizeHeadline, theme.T.Text, giode.FontWeightSemiBold), widgets.When(ag.Builtin, func() widgets.El { return widgets.Badge(tr("Built-in"), theme.T.Muted, theme.T.Surface2) })),
						widgets.Text(trn("{n} tools", len(ag.Tools))+" · "+trn("up to {n} steps", ag.MaxSteps), theme.SizeCaption, theme.T.Muted)),
					widgets.IconButton("agent/edit/"+ag.ID, "sliders-horizontal", tr("Edit agent"), 17, theme.T.Muted, func() { a.navigate("agents/edit/" + ag.ID) }),
				),
				widgets.Text(agentDescription(ag), theme.SizeFootnote, theme.T.Text2),
				widgets.Row(widgets.St{Gap: 6}, widgets.Text(tr("Start a task"), theme.SizeFootnote, ag.Color, giode.FontWeightMedium), widgets.Icon("arrow-up", 14, ag.Color)),
			)
		}, func() { a.newChat(store.ModeAgent, ag.ID) }))
	}
	return widgets.Col(widgets.St{Gap: 12, Width: -1},
		widgets.Text(tr("Agents plan multi-step tasks and carry them out with tools - searching, reading, running code and writing files to a private workspace you can share from."), theme.SizeFootnote, theme.T.Muted),
		widgets.Col(widgets.St{Gap: 10, Width: -1}, cards...),
	)
}
