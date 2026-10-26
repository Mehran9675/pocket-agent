package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) scrim(key string, onTap func()) widgets.El {
	return widgets.Tap(key, "Close", func(giode.PressState) widgets.El {
		return giode.Box(widgets.St{Width: -1, Height: -1, Background: theme.T.Scrim})
	}, onTap)
}
