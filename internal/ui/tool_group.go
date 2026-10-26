package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// toolGroup renders consecutive tool calls as a compact timeline.
func (a *App) toolGroup(c *store.Conversation, m *store.Message, runs []*store.ToolRun) widgets.El {
	var rows []widgets.El
	for _, t := range runs {
		if t.Name == "update_plan" && t.Status == store.ToolDone {
			continue // the plan card shows it
		}
		rows = append(rows, a.toolRow(c, m, t))
	}
	if len(rows) == 0 {
		return giode.Box(widgets.St{})
	}
	return widgets.Col(widgets.St{Gap: 2, Width: -1, Padding: giode.Inset{Left: 2}}, rows...)
}
