package ui

import (
	"runtime"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// openAttachSheet offers attachment sources.
func (a *App) openAttachSheet() {
	a.openSheet("attach", tr("Add to message"), func() widgets.El {
		return widgets.Col(widgets.St{Gap: 12, Width: -1},
			widgets.Group(
				widgets.ListRow("attach/photos", "image", "#38bdf8", tr("Photo library"), tr("Send images to vision models"), nil, func() {
					a.closeSheet()
					a.pickPhoto(giode.PhotoLibrary)
				}),
				widgets.When(runtime.GOOS == "android" || runtime.GOOS == "ios", func() widgets.El {
					return widgets.ListRow("attach/camera", "camera", "#f472b6", tr("Take a photo"), tr("Use the camera"), nil, func() {
						a.closeSheet()
						a.pickPhoto(giode.Camera)
					})
				}),
				widgets.ListRow("attach/files", "file-text", "#a78bfa", tr("Files"), tr("PDF, Word, Excel, PowerPoint, text, code, CSV…"), nil, func() {
					a.closeSheet()
					a.pickFiles()
				}),
				widgets.ListRow("attach/paste", "copy", "#34d399", tr("Paste from clipboard"), tr("Insert copied text"), nil, func() {
					a.closeSheet()
					a.win.ReadClipboard(func(text string) {
						a.composer.in.SetText(a.composer.in.Text() + text)
					})
				}),
			),
		)
	})
}
