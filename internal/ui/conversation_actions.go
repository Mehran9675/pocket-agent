package ui

import (
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// stop cancels c's running turn.
func (a *App) stop(c *store.Conversation) {
	if rs, ok := a.runs[c.ID]; ok {
		rs.cancel()
	}
}

// regenerate replaces the last assistant reply.
func (a *App) regenerate(c *store.Conversation) {
	if a.isRunning(c) || len(c.Messages) == 0 {
		return
	}
	if last := c.Messages[len(c.Messages)-1]; last.Role == llm.RoleAssistant {
		c.Messages = c.Messages[:len(c.Messages)-1]
	}
	a.startTurn(c)
}

// editMessage loads a user message back into the composer, dropping it
// and everything after.
func (a *App) editMessage(c *store.Conversation, m *store.Message) {
	if a.isRunning(c) {
		return
	}
	for i, x := range c.Messages {
		if x == m {
			c.Messages = c.Messages[:i]
			break
		}
	}
	c.Touch()
	a.st.SaveConversation(c)
	a.composer.load(m.Text, m.Attachments)
}

func (a *App) deleteConversation(c *store.Conversation) {
	a.stop(c)
	for i, x := range a.convs {
		if x == c {
			a.convs = append(a.convs[:i], a.convs[i+1:]...)
			break
		}
	}
	if a.current == c {
		a.current = nil
	}
	a.st.DeleteConversation(c)
}

func (a *App) togglePin(c *store.Conversation) {
	c.Pinned = !c.Pinned
	c.Version++
	store.SortConversations(a.convs)
	a.st.SaveConversation(c)
}
