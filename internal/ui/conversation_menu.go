package ui

import (
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// openConversationMenu shows actions for a conversation.
func (a *App) openConversationMenu(c *store.Conversation) {
	pinLabel := tr("Pin")
	if c.Pinned {
		pinLabel = tr("Unpin")
	}
	a.openSheet("convmenu", c.Title, func() widgets.El {
		return widgets.Col(widgets.St{Gap: 12, Width: -1},
			widgets.Group(
				widgets.ListRow("cm/rename", "pencil", theme.T.Primary, tr("Rename"), "", nil, func() { a.openRename(c) }),
				widgets.ListRow("cm/pin", "pin", theme.T.Primary, pinLabel, "", nil, func() { a.closeSheet(); a.togglePin(c) }),
				widgets.ListRow("cm/folder", "folder-input", theme.T.Primary, tr("Move to folder"), folderName(a, c), nil, func() { a.openMoveToFolder(c) }),
				widgets.ListRow("cm/system", "sliders-horizontal", theme.T.Primary, tr("Instructions for this chat"), tr("Override the system prompt"), nil, func() { a.openSystemPrompt(c) }),
			),
			widgets.Group(
				widgets.ListRow("cm/share", "share", "#34d399", tr("Share as text"), "", nil, func() {
					a.closeSheet()
					a.shareText(conversationMarkdown(c), c.Title)
				}),
				widgets.ListRow("cm/export", "download", "#34d399", tr("Export as Markdown"), "", nil, func() {
					a.closeSheet()
					a.exportConversation(c)
				}),
			),
			widgets.Group(
				widgets.ListRow("cm/delete", "trash-2", theme.T.Danger, tr("Delete conversation"), "", nil, func() {
					a.closeSheet()
					a.ask(tr("Delete conversation?"), tr("“{title}” and its files will be removed from this device.", "title", c.Title), tr("Delete"), true, func() {
						a.deleteConversation(c)
					})
				}),
			),
		)
	})
}
