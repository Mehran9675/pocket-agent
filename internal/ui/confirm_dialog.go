package ui

import (
	"github.com/mehran9675/giode"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// confirmState is a yes/no question.
type confirmState struct {
	title, message, action string
	danger                 bool
	onYes                  func()
}

// ask shows a confirmation; onYes runs if the user agrees.
func (a *App) ask(title, message, action string, danger bool, onYes func()) {
	a.confirm = &confirmState{title: title, message: message, action: action, danger: danger, onYes: onYes}
	a.confirmOpen = true
}

func (a *App) confirmView() widgets.El {
	if a.confirmPanel == nil {
		a.confirmPanel = newSheet(&a.confirmOpen, func() widgets.El {
			c := a.confirm
			if c == nil {
				return giode.Box(widgets.St{})
			}
			kind := widgets.BtnPrimary
			if c.danger {
				kind = widgets.BtnDanger
			}
			return widgets.Col(widgets.St{Gap: 18, Width: -1, Padding: giode.Inset{Bottom: 8}},
				widgets.Text(c.message, theme.SizeCallout, theme.T.Text2),
				widgets.Row(widgets.St{Gap: 10, Width: -1},
					widgets.Spacer(),
					widgets.Button("confirm/no", tr("Cancel"), "", widgets.BtnSecondary, func() { a.confirmOpen = false }),
					widgets.Button("confirm/yes", c.action, "", kind, func() {
						a.confirmOpen = false
						c.onYes()
					}),
				),
			)
		})
	}
	if c := a.confirm; c != nil {
		a.confirmPanel.Title(c.title).Styles(widgets.St{Background: theme.T.Elevated, Color: theme.T.Text})
	}
	return a.confirmPanel
}
