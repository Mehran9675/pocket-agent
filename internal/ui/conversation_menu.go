package ui

import (
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// openConversationMenu shows actions for a conversation.
func (a *App) openConversationMenu(c *store.Conversation) {
	pinLabel := "Pin"
	if c.Pinned {
		pinLabel = "Unpin"
	}
	a.openSheet("convmenu", c.Title, func() widgets.El {
		return widgets.Col(widgets.St{Gap: 12, Width: -1},
			widgets.Group(
				widgets.ListRow("cm/rename", "pencil", theme.T.Primary, "Rename", "", nil, func() { a.openRename(c) }),
				widgets.ListRow("cm/pin", "pin", theme.T.Primary, pinLabel, "", nil, func() { a.closeSheet(); a.togglePin(c) }),
				widgets.ListRow("cm/system", "sliders-horizontal", theme.T.Primary, "Instructions for this chat", "Override the system prompt", nil, func() { a.openSystemPrompt(c) }),
			),
			widgets.Group(
				widgets.ListRow("cm/share", "share", "#34d399", "Share as text", "", nil, func() {
					a.closeSheet()
					a.shareText(conversationMarkdown(c), c.Title)
				}),
				widgets.ListRow("cm/export", "download", "#34d399", "Export as Markdown", "", nil, func() {
					a.closeSheet()
					a.exportConversation(c)
				}),
			),
			widgets.Group(
				widgets.ListRow("cm/delete", "trash-2", theme.T.Danger, "Delete conversation", "", nil, func() {
					a.closeSheet()
					a.ask("Delete conversation?", "“"+c.Title+"” and its files will be removed from this device.", "Delete", true, func() {
						a.deleteConversation(c)
					})
				}),
			),
		)
	})
}
