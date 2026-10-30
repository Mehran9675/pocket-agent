package ui

import (
	"strconv"
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

var agentIcons = []string{"bot", "telescope", "code", "database", "pen-line", "flask-conical", "lightbulb", "wand-sparkles", "book-open", "hammer", "brain", "globe"}

var agentColors = []string{"#7c83ff", "#38bdf8", "#34d399", "#f59e0b", "#f472b6", "#fb7185", "#a78bfa", "#2dd4bf"}

func (a *App) agentEditor(id string, compact bool) widgets.El {
	ag := a.agentByID(id)
	if ag == nil {
		return a.agentsPage(compact)
	}
	return a.pageScaffold("agentedit/"+id, agentName(ag), compact,
		widgets.Button("agentedit/save/"+id, tr("Save"), "check", widgets.BtnPrimary, func() { a.saveAgent(ag) }),
		func() widgets.El { return a.agentEditorBody(ag) })
}

func (a *App) saveAgent(ag *store.Agent) {
	k := "agent/" + ag.ID + "/"
	if v := strings.TrimSpace(a.input(k+"name", "", ag.Name).Text()); v != "" {
		ag.Name = v
	}
	ag.Description = strings.TrimSpace(a.input(k+"desc", "", ag.Description).Text())
	ag.Instructions = strings.TrimSpace(a.input(k+"instr", "", ag.Instructions).Text())
	if n, err := strconv.Atoi(strings.TrimSpace(a.input(k+"steps", "", strconv.Itoa(ag.MaxSteps)).Text())); err == nil && n > 0 && n <= 100 {
		ag.MaxSteps = n
	}
	if n, err := strconv.Atoi(strings.TrimSpace(a.input(k+"budget", "", strconv.Itoa(ag.Budget()/1000)).Text())); err == nil && n > 0 {
		ag.TokenBudget = n
	}
	var starters []string
	for _, l := range strings.Split(a.input(k+"starters", "", strings.Join(ag.Starters, "\n")).Text(), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			starters = append(starters, l)
		}
	}
	ag.Starters = starters
	a.st.SaveAgents(a.agents)
	a.toast(giode.ToastSuccess, tr("Agent saved"))
}

func (a *App) agentEditorBody(ag *store.Agent) widgets.El {
	k := "agent/" + ag.ID + "/"
	name := a.input(k+"name", tr("Name"), ag.Name)
	desc := a.input(k+"desc", tr("What it's for"), ag.Description)
	instr := a.input(k+"instr", tr("Instructions"), ag.Instructions, func(i *input.Input) { i.Multiline(true) })
	steps := a.input(k+"steps", "20", strconv.Itoa(ag.MaxSteps), func(i *input.Input) { i.Keyboard(giode.KeyboardNumeric) })
	budget := a.input(k+"budget", strconv.Itoa(store.DefaultTokenBudget), strconv.Itoa(ag.Budget()/1000), func(i *input.Input) { i.Keyboard(giode.KeyboardNumeric) })
	starters := a.input(k+"starters", tr("One suggestion per line"), strings.Join(ag.Starters, "\n"), func(i *input.Input) { i.Multiline(true) })

	label := func(s string) widgets.El {
		return widgets.Text(s, theme.SizeFootnote, theme.T.Text2, giode.FontWeightMedium)
	}

	var icons []widgets.El
	for _, ic := range agentIcons {
		ic := ic
		icons = append(icons, widgets.Tap(k+"icon/"+ic, ic, func(s giode.PressState) widgets.El {
			border := "transparent"
			if ag.Icon == ic {
				border = ag.Color
			}
			return giode.Box(widgets.St{Width: 40, Height: 40, BorderRadius: 12, BorderWidth: 2, BorderColor: border, Background: widgets.HoverBg(s, theme.T.Surface2), Align: giode.AlignCenter, Justify: giode.JustifyCenter},
				widgets.Icon(ic, 20, ag.Color))
		}, func() { ag.Icon = ic; a.st.SaveAgents(a.agents) }))
	}
	var colors []widgets.El
	for _, c := range agentColors {
		c := c
		colors = append(colors, widgets.Tap(k+"color/"+c, c, func(s giode.PressState) widgets.El {
			border := "transparent"
			if ag.Color == c {
				border = theme.T.Text
			}
			return giode.Box(widgets.St{Width: 32, Height: 32, BorderRadius: 16, BorderWidth: 2, BorderColor: border, Padding: giode.UniformInset(3)},
				giode.Box(widgets.St{Width: -1, Height: -1, BorderRadius: 13, Background: c}))
		}, func() { ag.Color = c; a.st.SaveAgents(a.agents) }))
	}

	enabled := map[string]bool{}
	for _, t := range ag.Tools {
		enabled[t] = true
	}
	setTool := func(name string, on bool) {
		var out []string
		for _, t := range ag.Tools {
			if t != name {
				out = append(out, t)
			}
		}
		if on {
			out = append(out, name)
		}
		ag.Tools = out
		a.st.SaveAgents(a.agents)
	}
	var toolRows []widgets.El
	for _, t := range tools.Builtins() {
		t := t
		toolRows = append(toolRows, widgets.ListRow(k+"tool/"+t.Name, t.Icon, ag.Color, tr(t.Title), toolSummary(t), widgets.Toggle(k+"tt/"+t.Name, enabled[t.Name], func(v bool) { setTool(t.Name, v) }), nil))
	}

	kids := []widgets.El{
		widgets.Row(widgets.St{Gap: 14, Width: -1}, widgets.IconTile(ag.Icon, ag.Color, 56), widgets.Col(widgets.St{Gap: 4, FlexGrow: 1}, label(tr("Name")), name)),
		widgets.Col(widgets.St{Gap: 6, Width: -1}, label(tr("Description")), desc),
		widgets.Col(widgets.St{Gap: 6, Width: -1}, label(tr("Icon")), giode.Row(widgets.St{Gap: 8, Wrap: giode.WrapWrap}, icons...)),
		widgets.Col(widgets.St{Gap: 6, Width: -1}, label(tr("Color")), giode.Row(widgets.St{Gap: 8, Wrap: giode.WrapWrap}, colors...)),
		widgets.Col(widgets.St{Gap: 6, Width: -1}, label(tr("Instructions")), widgets.Text(tr("The agent's system prompt: its role, method and what a finished job looks like."), theme.SizeCaption, theme.T.Muted), giode.Box(widgets.St{Width: -1, MinHeight: 160}, instr)),
		widgets.Col(widgets.St{Gap: 6, Width: -1}, label(tr("Conversation starters")), giode.Box(widgets.St{Width: -1, MinHeight: 90}, starters)),
		widgets.Col(widgets.St{Gap: 6, Width: -1}, label(tr("Maximum steps")), widgets.Text(tr("How many model calls the agent may make for one task."), theme.SizeCaption, theme.T.Muted), giode.Box(widgets.St{Width: 120}, steps)),
		widgets.Col(widgets.St{Gap: 6, Width: -1}, label(tr("Token budget (thousands)")), widgets.Text(tr("The agent pauses once a task has used this many tokens; Continue resumes it."), theme.SizeCaption, theme.T.Muted), giode.Box(widgets.St{Width: 120}, budget)),
		widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("Tools")), widgets.Group(toolRows...)),
		widgets.Group(widgets.ListRow(k+"auto", "shield-check", theme.T.Warning, tr("Run without asking"), tr("Skip approval for actions like deleting files and MCP writes"), widgets.Toggle(k+"autot", ag.AutoApprove, func(v bool) {
			ag.AutoApprove = v
			a.st.SaveAgents(a.agents)
		}), nil)),
	}
	if ag.Builtin {
		kids = append(kids, widgets.Text(tr("Built-in agents can be customized; changes to built-ins aren't saved across reinstalls. Duplicate one to make your own."), theme.SizeCaption, theme.T.Muted),
			widgets.Button(k+"dup", tr("Duplicate"), "copy", widgets.BtnSecondary, func() {
				cp := *ag
				cp.ID, cp.Builtin, cp.Name = store.NewID(), false, tr("{name} copy", "name", agentName(ag))
				cp.Tools = append([]string(nil), ag.Tools...)
				a.agents = append(a.agents, &cp)
				a.st.SaveAgents(a.agents)
				a.navigate("agents/edit/" + cp.ID)
			}))
	} else {
		kids = append(kids, widgets.Button(k+"delete", tr("Delete agent"), "trash-2", widgets.BtnDanger, func() {
			a.ask(tr("Delete {name}?", "name", agentName(ag)), tr("Conversations with it are kept."), tr("Delete"), true, func() {
				for i, x := range a.agents {
					if x == ag {
						a.agents = append(a.agents[:i], a.agents[i+1:]...)
						break
					}
				}
				a.st.SaveAgents(a.agents)
				a.back()
			})
		}))
	}
	kids = append(kids, widgets.Button(k+"start", tr("Start a task with {name}", "name", agentName(ag)), "play", widgets.BtnPrimary, func() {
		a.saveAgent(ag)
		a.newChat(store.ModeAgent, ag.ID)
	}, widgets.St{Width: -1}))
	return widgets.Col(widgets.St{Gap: 18, Width: -1}, kids...)
}
