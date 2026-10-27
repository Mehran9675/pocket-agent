package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) mcpBody() widgets.El {
	var rows []widgets.El
	for i := range a.settings.MCPServers {
		srv := &a.settings.MCPServers[i]
		status := a.mcpStatus[srv.ID]
		if !srv.Enabled {
			status = "Disabled"
		}
		id := srv.ID
		rows = append(rows, widgets.ListRow("mcp/"+id, "plug", "#f472b6", srv.Name, firstNonEmpty(status, srv.URL), widgets.Row(widgets.St{Gap: 4},
			widgets.Toggle("mcp/t/"+id, srv.Enabled, func(v bool) {
				srv.Enabled = v
				a.saveSettings()
				if v {
					a.connectServer(*srv)
				} else {
					delete(a.mcpTools, id)
					delete(a.mcp, id)
				}
			}),
			widgets.IconButton("mcp/rm/"+id, "trash-2", "Remove", 16, theme.T.Muted, func() {
				a.ask("Remove "+srv.Name+"?", "Its tools will no longer be available.", "Remove", true, func() {
					for j, s := range a.settings.MCPServers {
						if s.ID == id {
							a.settings.MCPServers = append(a.settings.MCPServers[:j], a.settings.MCPServers[j+1:]...)
							break
						}
					}
					delete(a.mcpTools, id)
					delete(a.mcp, id)
					a.saveSettings()
				})
			})), nil))
	}
	nameIn := a.input("mcp/name", "Name, e.g. GitHub", "")
	urlIn := a.input("mcp/url", "https://example.com/mcp", "", func(i *input.Input) { i.Keyboard(giode.KeyboardURL) })
	authIn := a.input("mcp/auth", "Auth header (optional)", "", func(i *input.Input) { i.Password(true) })
	add := func() {
		u := strings.TrimSpace(urlIn.Text())
		if !strings.HasPrefix(u, "http") {
			a.toast(giode.ToastWarning, "Enter the server's http(s) URL")
			return
		}
		srv := tools.MCPServer{ID: store.NewID(), Name: firstNonEmpty(strings.TrimSpace(nameIn.Text()), "MCP server"), URL: u, Enabled: true}
		if h := strings.TrimSpace(authIn.Text()); h != "" {
			srv.Headers = map[string]string{"Authorization": h}
		}
		a.settings.MCPServers = append(a.settings.MCPServers, srv)
		a.saveSettings()
		a.connectServer(srv)
		nameIn.SetText("")
		urlIn.SetText("")
		authIn.SetText("")
	}
	kids := []widgets.El{
		widgets.Text("Connect remote Model Context Protocol servers (Streamable HTTP) to give chats and agents new tools: GitHub, Notion, Linear, your own APIs.", theme.SizeFootnote, theme.T.Muted),
	}
	if len(rows) > 0 {
		kids = append(kids, widgets.Group(rows...))
	}
	kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel("Add a server"), widgets.Card(widgets.St{Gap: 12},
		nameIn, urlIn, authIn,
		widgets.Button("mcp/add", "Connect server", "plus", widgets.BtnPrimary, add))))
	for _, srv := range a.settings.MCPServers {
		ts := a.mcpTools[srv.ID]
		if len(ts) == 0 {
			continue
		}
		var trows []widgets.El
		for _, t := range ts {
			trows = append(trows, widgets.ListRow("mcptool/"+t.Name, "wrench", "#f472b6", t.Title, firstSentence(strings.TrimPrefix(t.Description, "["+srv.Name+"] ")), nil, nil))
		}
		kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(srv.Name+" tools"), widgets.Group(trows...)))
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1}, kids...)
}
