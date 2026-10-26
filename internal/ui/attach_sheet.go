package ui

import (
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// openAttachSheet offers attachment sources.
func (a *App) openAttachSheet() {
	a.openSheet("attach", "Add to message", func() widgets.El {
		return widgets.Col(widgets.St{Gap: 12, Width: -1},
			widgets.Group(
				widgets.ListRow("attach/photos", "image", "#38bdf8", "Photos & images", "Send images to vision models", nil, func() {
					a.closeSheet()
					a.pickFiles(true)
				}),
				widgets.ListRow("attach/files", "file-text", "#a78bfa", "Files", "PDF, Word, Excel, PowerPoint, text, code, CSV…", nil, func() {
					a.closeSheet()
					a.pickFiles(false)
				}),
				widgets.ListRow("attach/paste", "copy", "#34d399", "Paste from clipboard", "Insert copied text", nil, func() {
					a.closeSheet()
					a.win.ReadClipboard(func(text string) {
						a.composer.in.SetText(a.composer.in.Text() + text)
					})
				}),
			),
		)
	})
}
