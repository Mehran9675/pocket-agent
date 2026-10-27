package ui

import (
	"context"
	"time"

	"gioui.org/unit"

	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// markdown renders text as Markdown; key identifies it for selection.
func (a *App) markdown(key, text string) widgets.El {
	scale := a.settings.FontScale
	if scale == 0 {
		scale = 1
	}
	size := float32(theme.SizeBody * scale)
	return giode.Markdown(text, giode.MarkdownOptions{
		Styles:         widgets.St{FontSize: unit.Sp(size), LineHeight: pr.LineHeight(size * 1.5), Color: theme.T.Text},
		CodeFont:       theme.FontMono,
		CodeBackground: theme.T.CodeBg,
		LinkColor:      theme.T.PrimaryText,
		OnLink:         a.openLink,
		Selectable:     key,
		Gap:            12,
	})
}

// openLink opens url in the browser, or copies it when it can't.
func (a *App) openLink(u string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.win.OpenURL(ctx, u); err != nil {
			a.Post(func() { a.copyText(u) })
		}
	}()
}
