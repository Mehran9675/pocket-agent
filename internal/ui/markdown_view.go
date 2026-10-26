package ui

import (
	"github.com/mehran9675/pocket-agent/internal/platform"
	"github.com/mehran9675/pocket-agent/internal/ui/markdown"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) markdown(key, text string) widgets.El {
	doc := a.docs.Get(key, func() *markdown.Doc {
		return &markdown.Doc{Key: key, OnLink: a.openLink, OnCopy: a.copyText}
	})
	doc.Set(text)
	scale := a.settings.FontScale
	if scale == 0 {
		scale = 1
	}
	return doc.Element(float32(theme.SizeBody * scale))
}

// openLink opens url in the browser, or copies it.
func (a *App) openLink(u string) {
	if err := platform.OpenURL(u); err != nil {
		a.copyText(u)
	}
}
