package ui

import (
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) openDefaultModelSheet() {
	// Reuse the picker, then copy the choice into the default.
	prevCur := a.current
	a.current = nil
	saved := a.draft.model
	a.draft.model = a.settings.DefaultModel
	a.openModelSheet()
	a.sheet.title = tr("Default model")
	inner := a.sheet.body
	a.sheet.body = func() widgets.El {
		if a.draft.model != a.settings.DefaultModel {
			a.settings.DefaultModel = a.draft.model
			a.saveSettings()
		}
		return inner()
	}
	a.current = prevCur
	if prevCur != nil {
		a.draft.model = saved
	}
}
