package ui

import (
	"strings"

	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) settingsPage(sub string, compact bool) widgets.El {
	switch {
	case sub == "providers":
		return a.pageScaffold("providers", tr("Model providers"), compact, nil, a.providersBody)
	case strings.HasPrefix(sub, "provider/"):
		id := strings.TrimPrefix(sub, "provider/")
		return a.pageScaffold("provider/"+id, a.providerTitle(id), compact, nil, func() widgets.El { return a.providerBody(id) })
	case sub == "search":
		return a.pageScaffold("search", tr("Web search"), compact, nil, a.searchBody)
	case sub == "tools":
		return a.pageScaffold("tools", tr("Tools & permissions"), compact, nil, a.toolsBody)
	case sub == "mcp":
		return a.pageScaffold("mcp", tr("MCP servers"), compact, nil, a.mcpBody)
	case sub == "skills":
		return a.pageScaffold("skills", tr("Skills"), compact, nil, a.skillsBody)
	case strings.HasPrefix(sub, "skill/"):
		name := strings.TrimPrefix(sub, "skill/")
		return a.pageScaffold("skill/"+name, skillPageTitle(name), compact, nil, func() widgets.El { return a.skillEditorBody(name) })
	case sub == "memory":
		return a.pageScaffold("memory", tr("Memory"), compact, nil, a.memoryBody)
	case sub == "instructions":
		return a.pageScaffold("instructions", tr("Custom instructions"), compact, nil, a.instructionsBody)
	case sub == "local/perf":
		return a.pageScaffold("local/perf", tr("Performance & memory"), compact, nil, a.enginePerfBody)
	case sub == "local/sampling":
		return a.pageScaffold("local/sampling", tr("Sampling"), compact, nil, a.samplingBody)
	case sub == "local":
		return a.pageScaffold("local", tr("On-device engine"), compact, nil, a.engineBody)
	case sub == "about":
		return a.pageScaffold("about", tr("About"), compact, nil, a.aboutBody)
	}
	return a.pageScaffold("settings", tr("Settings"), compact, nil, a.settingsHome)
}
