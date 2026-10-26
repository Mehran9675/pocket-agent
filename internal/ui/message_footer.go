package ui

import (
	"fmt"
	"strings"

	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) messageFooter(c *store.Conversation, m *store.Message, last bool) widgets.El {
	text := m.ReplyText()
	kids := []widgets.El{
		widgets.IconButton("a/copy/"+m.ID, "copy", "Copy", 15, theme.T.Faint, func() { a.copyText(text) }),
		widgets.IconButton("a/share/"+m.ID, "share", "Share", 15, theme.T.Faint, func() { a.shareText(text, c.Title) }),
	}
	if last && !a.isRunning(c) {
		kids = append(kids, widgets.IconButton("a/regen/"+m.ID, "refresh-cw", "Regenerate", 15, theme.T.Faint, func() { a.regenerate(c) }))
	}
	var meta []string
	if m.Model != "" {
		meta = append(meta, m.Model)
	}
	if m.Usage.OutputTokens > 0 {
		meta = append(meta, fmt.Sprintf("%s tokens", compactNum(m.Usage.InputTokens+m.Usage.OutputTokens)))
	}
	if m.Duration > 0 {
		meta = append(meta, fmtDuration(m.Duration))
	}
	kids = append(kids, widgets.Gap(6, 0), widgets.Line(strings.Join(meta, " · "), theme.SizeCaption, theme.T.Faint))
	return widgets.Row(widgets.St{Gap: 0, Width: -1}, kids...)
}
