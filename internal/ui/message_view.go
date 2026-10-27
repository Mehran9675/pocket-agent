package ui

import (
	"fmt"
	"strings"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// messageRows lays a message out as rows for the message list.
func (a *App) messageRows(c *store.Conversation, m *store.Message, last bool) []widgets.VRow {
	if m.Role == llm.RoleUser {
		token := fmt.Sprint(textToken(m.Text), len(m.Attachments), m.Version, a.isRunning(c), a.textEnv())
		return []widgets.VRow{{Key: m.ID, Token: token, Build: func() widgets.El { return a.userMessage(c, m) }}}
	}
	return a.assistantRows(c, m, last)
}

// expandedSig summarizes which parts of m are expanded, for row tokens.
func (a *App) expandedSig(m *store.Message) string {
	var b strings.Builder
	for k, v := range a.expanded {
		if v && strings.HasPrefix(k, m.ID) {
			b.WriteString(k)
		}
	}
	return b.String()
}
