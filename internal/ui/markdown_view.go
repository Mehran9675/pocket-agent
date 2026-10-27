package ui

import (
	"context"
	"fmt"
	"time"

	"gioui.org/unit"

	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// markdown renders text as Markdown; key identifies it for selection.
// Long text is rendered in separately memoized chunks (see chunkSize).
func (a *App) markdown(key, text string) widgets.El {
	scale := a.settings.FontScale
	if scale == 0 {
		scale = 1
	}
	size := float32(theme.SizeBody * scale)
	chunks := splitMarkdown(text, chunkSize)
	if len(chunks) == 1 {
		return a.markdownChunk(key, text, size)
	}
	kids := make([]widgets.El, len(chunks))
	for i, chunk := range chunks {
		ckey := fmt.Sprintf("%s/%d", key, i)
		kids[i] = giode.Memo("md/"+ckey, func() widgets.El { return a.markdownChunk(ckey, chunk, size) }, chunk, size)
	}
	return widgets.Col(widgets.St{Gap: 12, Width: -1}, kids...)
}

func (a *App) markdownChunk(key, text string, size float32) widgets.El {
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
