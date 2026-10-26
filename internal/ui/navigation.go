package ui

import (
	"github.com/mehran9675/pocket-agent/internal/store"
)

func (a *App) navigate(route string) {
	if a.route == route {
		return
	}
	a.history = append(a.history, a.route)
	a.route = route
	a.drawerOpen = false
}

// navigateRoot replaces the history.
func (a *App) navigateRoot(route string) {
	a.history = nil
	a.route = route
	a.drawerOpen = false
}

func (a *App) back() {
	if n := len(a.history); n > 0 {
		a.route = a.history[n-1]
		a.history = a.history[:n-1]
		return
	}
	a.route = ""
}

// openConversation shows c in the chat.
func (a *App) openConversation(c *store.Conversation) {
	a.current = c
	a.navigateRoot("")
	a.chat.jumpToBottom = true
	a.composer.reset()
}

// newChat starts a fresh draft.
func (a *App) newChat(mode store.Mode, agentID string) {
	a.current = nil
	a.draft.mode = mode
	a.draft.agentID = agentID
	if mode == store.ModeAgent && agentID == "" && len(a.agents) > 0 {
		a.draft.agentID = a.agents[0].ID
	}
	a.navigateRoot("")
	a.composer.reset()
	a.composer.focus()
}
