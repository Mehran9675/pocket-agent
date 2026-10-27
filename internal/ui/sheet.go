package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/sheet"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// sheetSpec describes the open sheet: a bottom sheet on phones, a
// centered dialog on wider screens (Giode's Sheet, Adaptive).
type sheetSpec struct {
	key    string
	title  string
	body   func() widgets.El
	footer func() widgets.El // pinned below the scrolling body
}

// openSheet shows a sheet; it returns the spec so a footer can be set.
func (a *App) openSheet(key, title string, body func() widgets.El) *sheetSpec {
	a.sheet = &sheetSpec{key: key, title: title, body: body}
	a.sheetOpen = true
	return a.sheet
}

func (a *App) closeSheet() { a.sheetOpen = false }

// sheetView is the one Sheet the app shows its sheets in.
func (a *App) sheetView() widgets.El {
	if a.sheetPanel == nil {
		a.sheetPanel = giode.Sheet(&a.sheetOpen, func() widgets.El {
			if a.sheet == nil {
				return giode.Box(widgets.St{})
			}
			return a.sheet.body()
		}).Adaptive(true)
	}
	s := a.sheet
	if s == nil {
		return a.sheetPanel
	}
	a.sheetPanel.Title(s.title).Footer(s.footer).
		Styles(widgets.St{Background: theme.T.Elevated, Color: theme.T.Text})
	return a.sheetPanel
}

// newSheet is for sheets with their own open state.
func newSheet(open *bool, content func() widgets.El) *sheet.Sheet {
	return giode.Sheet(open, content).Adaptive(true)
}
