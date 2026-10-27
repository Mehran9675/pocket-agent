package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// View is the root element.
func (a *App) View() widgets.El {
	body := giode.Responsive(
		giode.Breakpoint{MaxWidth: 699, Name: "compact", View: a.compactView},
		giode.Breakpoint{MaxWidth: 1149, Name: "medium", View: a.wideView},
		giode.Breakpoint{Name: "expanded", View: a.wideView},
	)
	if a.route != "" {
		giode.HandleBack(a.back)
	}
	return giode.Box(widgets.St{Width: -1, Height: -1, Background: theme.T.Bg, FlexDirection: giode.FlexDirectionColumn},
		body,
		a.overlays(),
		a.toasts,
	)
}

func (a *App) compactView() widgets.El {
	var page widgets.El
	if a.route == "" {
		page = a.chat.view(true)
	} else {
		page = a.pageView(a.route, true)
	}
	return giode.Box(widgets.St{Width: -1, Height: -1, FlexDirection: giode.FlexDirectionColumn},
		page,
		a.drawer, // an overlay: takes no space
	)
}

func (a *App) wideView() widgets.El {
	expanded := giode.ActiveBreakpoint() == "expanded"
	var main widgets.El
	if a.route == "" {
		main = a.chat.view(false)
	} else {
		main = a.pageView(a.route, false)
	}
	kids := []widgets.El{
		giode.Box(widgets.St{Width: 296, Height: -1, Background: theme.T.Sidebar}, a.sidebar.view(false)),
		widgets.VDivider(),
		giode.Box(widgets.St{FlexGrow: 1, Height: -1, FlexDirection: giode.FlexDirectionColumn}, main),
	}
	if expanded && a.route == "" && a.inspectorRelevant() {
		kids = append(kids, widgets.VDivider(), giode.Box(widgets.St{Width: 340, Height: -1, Background: theme.T.Sidebar}, a.inspector()))
	}
	return widgets.Row(widgets.St{Width: -1, Height: -1, Align: giode.AlignStart}, kids...)
}

// pageView renders a non-chat route.
func (a *App) pageView(route string, compact bool) widgets.El {
	parts := strings.SplitN(route, "/", 3)
	switch parts[0] {
	case "settings":
		sub := ""
		if len(parts) > 1 {
			sub = strings.Join(parts[1:], "/")
		}
		return a.settingsPage(sub, compact)
	case "agents":
		if len(parts) > 2 && parts[1] == "edit" {
			return a.agentEditor(parts[2], compact)
		}
		return a.agentsPage(compact)
	case "library":
		return a.libraryPage(compact)
	case "file":
		return a.fileViewer(strings.TrimPrefix(route, "file/"), compact)
	}
	return a.chat.view(compact)
}
