package ui

import (
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) toolsBody() widgets.El {
	inChat := map[string]bool{}
	for _, n := range a.settings.ChatTools {
		inChat[n] = true
	}
	groups := map[string][]*tools.Tool{}
	var order []string
	for _, t := range tools.Builtins() {
		if t.Name == "web_search" || t.Name == "fetch_url" || t.Name == "update_plan" {
			continue // web is the composer's toggle; plans are agent-only
		}
		if _, ok := groups[t.Group]; !ok {
			order = append(order, t.Group)
		}
		groups[t.Group] = append(groups[t.Group], t)
	}
	kids := []widgets.El{widgets.Text(tr("Choose the tools models may use in regular chats. Agents use the tools set in each agent; web search is switched on per chat with the globe button."), theme.SizeFootnote, theme.T.Muted)}
	for _, g := range order {
		var rows []widgets.El
		for _, t := range groups[g] {
			t := t
			rows = append(rows, widgets.ListRow("tool/"+t.Name, t.Icon, "#a78bfa", tr(t.Title), toolSummary(t), widgets.Toggle("tool/t/"+t.Name, inChat[t.Name], func(v bool) {
				a.setChatTool(t.Name, v)
			}), nil))
		}
		kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr(g)), widgets.Group(rows...)))
	}
	// Approvals
	var apRows []widgets.El
	approvalTools := []*tools.Tool{}
	for _, t := range tools.Builtins() {
		if t.Approval {
			approvalTools = append(approvalTools, t)
		}
	}
	for _, ts := range a.mcpTools {
		for _, t := range ts {
			if t.Approval {
				approvalTools = append(approvalTools, t)
			}
		}
	}
	for _, t := range approvalTools {
		t := t
		apRows = append(apRows, widgets.ListRow("ap/"+t.Name, t.Icon, theme.T.Warning, t.Title, ifStr(a.settings.AutoApprove[t.Name], tr("Runs without asking"), tr("Asks before running")), widgets.Toggle("ap/t/"+t.Name, !a.settings.AutoApprove[t.Name], func(v bool) {
			if v {
				delete(a.settings.AutoApprove, t.Name)
			} else {
				a.settings.AutoApprove[t.Name] = true
			}
			a.saveSettings()
		}), nil))
	}
	if len(apRows) > 0 {
		kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("Ask before running")), widgets.Group(apRows...)))
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1}, kids...)
}

func (a *App) setChatTool(name string, on bool) {
	out := a.settings.ChatTools[:0:0]
	for _, n := range a.settings.ChatTools {
		if n != name {
			out = append(out, n)
		}
	}
	if on {
		out = append(out, name)
	}
	a.settings.ChatTools = out
	a.saveSettings()
}
