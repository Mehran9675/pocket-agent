package ui

import (
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// openModeSheet picks chat, agent or research mode.
func (a *App) openModeSheet() {
	set := func(mode store.Mode, agentID string) {
		a.closeSheet()
		if c := a.current; c != nil {
			if c.Mode == mode && c.AgentID == agentID {
				return
			}
			if len(c.Messages) > 0 {
				// A different mode starts a fresh conversation.
				a.newChat(mode, agentID)
				return
			}
		}
		a.draft.mode, a.draft.agentID = mode, agentID
	}
	cur, curAgent := a.draft.mode, a.draft.agentID
	if a.current != nil {
		cur, curAgent = a.current.Mode, a.current.AgentID
	}
	check := func(on bool) widgets.El {
		if on {
			return widgets.Icon("check", 18, theme.T.Primary)
		}
		return nil
	}
	a.openSheet("mode", "Mode", func() widgets.El {
		agentRows := []widgets.El{}
		for _, ag := range a.agents {
			ag := ag
			agentRows = append(agentRows, widgets.ListRow("mode/agent/"+ag.ID, ag.Icon, ag.Color, ag.Name, ag.Description,
				check(cur == store.ModeAgent && curAgent == ag.ID), func() { set(store.ModeAgent, ag.ID) }))
		}
		return widgets.Col(widgets.St{Gap: 14, Width: -1},
			widgets.Group(
				widgets.ListRow("mode/chat", "message-square", theme.T.Primary, "Chat", "Conversation with optional tools and web search", check(cur == store.ModeChat), func() { set(store.ModeChat, "") }),
				widgets.ListRow("mode/research", "telescope", "#38bdf8", "Deep research", "Searches and reads dozens of sources, then writes a cited report", check(cur == store.ModeResearch), func() { set(store.ModeResearch, "") }),
			),
			widgets.SectionLabel("Agents"),
			widgets.Group(agentRows...),
			widgets.Button("mode/manage", "Manage agents", "settings", widgets.BtnSecondary, func() {
				a.closeSheet()
				a.navigate("agents")
			}, widgets.St{Width: -1}),
		)
	})
}
