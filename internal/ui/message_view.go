package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) messageView(c *store.Conversation, m *store.Message, last bool) widgets.El {
	if m.Role == llm.RoleUser {
		return a.userMessage(c, m)
	}
	// Finished replies are cached; they rebuild only when they change.
	if !m.Streaming {
		return giode.Memo("msg/"+m.ID, func() widgets.El { return a.assistantMessage(c, m, last) },
			m.Version, last, len(m.Blocks), theme.T.Dark, a.expandedSig(m), a.isRunning(c))
	}
	return a.assistantMessage(c, m, last)
}

// expandedSig summarizes which parts of m are expanded, for memo deps.
func (a *App) expandedSig(m *store.Message) string {
	var b strings.Builder
	for k, v := range a.expanded {
		if v && strings.HasPrefix(k, m.ID) {
			b.WriteString(k)
		}
	}
	return b.String()
}
