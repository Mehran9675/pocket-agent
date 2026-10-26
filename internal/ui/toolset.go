package ui

import (
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

// toolSet assembles the tools for a turn.
func (a *App) toolSet(c *store.Conversation, ag *store.Agent) *tools.Set {
	set := tools.NewSet()
	add := func(name string) {
		if t, ok := tools.Builtin(name); ok {
			if (name == "remember" || name == "forget") && !a.settings.MemoryEnabled {
				return
			}
			set.Add(t)
			return
		}
		for _, ts := range a.mcpTools {
			for _, t := range ts {
				if t.Name == name {
					set.Add(t)
				}
			}
		}
	}
	switch c.Mode {
	case store.ModeAgent:
		if ag != nil {
			for _, n := range ag.Tools {
				add(n)
			}
		}
		// Agents also get every connected MCP tool.
		for _, ts := range a.mcpTools {
			for _, t := range ts {
				set.Add(t)
			}
		}
	default:
		for _, n := range a.settings.ChatTools {
			add(n)
		}
		if c.WebSearch {
			add("web_search")
			add("fetch_url")
		}
		for _, srv := range a.settings.MCPServers {
			if srv.Enabled {
				for _, t := range a.mcpTools[srv.ID] {
					set.Add(t)
				}
			}
		}
	}
	return set
}
