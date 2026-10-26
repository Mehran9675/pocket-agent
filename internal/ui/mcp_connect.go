package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/mehran9675/pocket-agent/internal/tools"
)

// connectMCP connects the enabled MCP servers and lists their tools.
func (a *App) connectMCP() {
	for _, srv := range a.settings.MCPServers {
		if srv.Enabled {
			a.connectServer(srv)
		}
	}
}

func (a *App) connectServer(srv tools.MCPServer) {
	cl := tools.NewMCPClient(srv)
	a.mcp[srv.ID] = cl
	a.mcpStatus[srv.ID] = "Connecting…"
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ts, err := cl.Tools(ctx)
		a.Post(func() {
			if a.mcp[srv.ID] != cl {
				return
			}
			if err != nil {
				a.mcpStatus[srv.ID] = "Error: " + err.Error()
				delete(a.mcpTools, srv.ID)
				return
			}
			a.mcpTools[srv.ID] = ts
			a.mcpStatus[srv.ID] = fmt.Sprintf("Connected · %d tools", len(ts))
		})
	}()
}
