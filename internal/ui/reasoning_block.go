package ui

import (
	"fmt"
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) reasoningBlock(m *store.Message, idx int, text string, active bool) widgets.El {
	key := fmt.Sprintf("%s/think/%d", m.ID, idx)
	open := a.expanded[key]
	label := "Thought process"
	if active {
		label = "Thinking…"
	}
	header := widgets.Tap(key, label, func(s giode.PressState) widgets.El {
		lead := widgets.Icon("brain", 15, theme.T.Muted)
		if active {
			lead = giode.Spinner(widgets.St{Width: 14, Height: 14, Color: theme.T.Muted})
		}
		chev := "chevron-right"
		if open {
			chev = "chevron-down"
		}
		return widgets.Row(widgets.St{Gap: 8, Padding: giode.SymmetricInset(10, 6), BorderRadius: 10, Background: widgets.HoverBg(s, "transparent")},
			lead, widgets.Text(label, theme.SizeFootnote, theme.T.Muted, giode.FontWeightMedium), widgets.Icon(chev, 14, theme.T.Faint))
	}, func() { a.expanded[key] = !open })
	if !open {
		if active {
			// Show the latest line while thinking.
			tail := strings.TrimSpace(text)
			if i := strings.LastIndex(tail, "\n"); i >= 0 {
				tail = tail[i+1:]
			}
			return widgets.Col(widgets.St{Gap: 2}, header, giode.Box(widgets.St{Padding: giode.Inset{Left: 12}}, widgets.Line(tail, theme.SizeFootnote, theme.T.Faint)))
		}
		return header
	}
	return widgets.Col(widgets.St{Gap: 6, Width: -1}, header,
		widgets.Row(widgets.St{Gap: 12, Align: giode.AlignStart, Padding: giode.Inset{Left: 12}},
			giode.Box(widgets.St{Width: 2, Height: -1, MinHeight: 16, Background: theme.T.Border}),
			giode.Box(widgets.St{FlexGrow: 1}, plainChunks(key, strings.TrimSpace(text), widgets.St{FontSize: theme.SizeFootnote, Color: theme.T.Muted, LineHeight: 20}, false))))
}

// plainChunks renders long plain text in separately memoized chunks
// (see chunkSize), optionally selectable.
func plainChunks(key, text string, st widgets.St, selectable bool) widgets.El {
	chunks := splitPlain(text, chunkSize)
	render := func(i int, s string) widgets.El {
		t := giode.Text(s, st)
		if selectable {
			t = t.Selectable(fmt.Sprintf("%s/%d", key, i))
		}
		return t
	}
	if len(chunks) == 1 {
		return render(0, text)
	}
	kids := make([]widgets.El, len(chunks))
	for i, chunk := range chunks {
		kids[i] = giode.Memo(fmt.Sprintf("txt/%s/%d", key, i), func() widgets.El { return render(i, chunk) }, chunk)
	}
	return widgets.Col(widgets.St{Width: -1}, kids...)
}
