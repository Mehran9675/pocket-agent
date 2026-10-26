package ui

import (
	"strings"

	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) settingsPage(sub string, compact bool) widgets.El {
	switch {
	case sub == "providers":
		return a.pageScaffold("providers", "Model providers", compact, nil, a.providersBody)
	case strings.HasPrefix(sub, "provider/"):
		id := strings.TrimPrefix(sub, "provider/")
		return a.pageScaffold("provider/"+id, a.providerTitle(id), compact, nil, func() widgets.El { return a.providerBody(id) })
	case sub == "search":
		return a.pageScaffold("search", "Web search", compact, nil, a.searchBody)
	case sub == "tools":
		return a.pageScaffold("tools", "Tools & permissions", compact, nil, a.toolsBody)
	case sub == "mcp":
		return a.pageScaffold("mcp", "MCP servers", compact, nil, a.mcpBody)
	case sub == "memory":
		return a.pageScaffold("memory", "Memory", compact, nil, a.memoryBody)
	case sub == "instructions":
		return a.pageScaffold("instructions", "Custom instructions", compact, nil, a.instructionsBody)
	case sub == "local":
		return a.pageScaffold("local", "On-device engine", compact, nil, a.engineBody)
	case sub == "about":
		return a.pageScaffold("about", "About", compact, nil, a.aboutBody)
	}
	return a.pageScaffold("settings", "Settings", compact, nil, a.settingsHome)
}
