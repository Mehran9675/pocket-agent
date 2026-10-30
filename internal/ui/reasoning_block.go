package ui

import (
	"fmt"
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// reasoningRows adds a thinking block's rows: its header (with the
// latest line while thinking) and, when expanded, the text in chunks.
func (a *App) reasoningRows(m *store.Message, idx int, text string, active bool, add, addText func(key, token string, build func() widgets.El)) {
	key := fmt.Sprintf("%s/think/%d", m.ID, idx)
	open := a.expanded[key]
	label := tr("Thought process")
	if active {
		label = tr("Thinking…")
	}
	tail := ""
	if active && !open {
		tail = strings.TrimSpace(text)
		if i := strings.LastIndex(tail, "\n"); i >= 0 {
			tail = tail[i+1:]
		}
	}
	add(fmt.Sprintf("think%d", idx), fmt.Sprint(open, active, tail), func() widgets.El {
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
		if tail == "" {
			return header
		}
		return widgets.Col(widgets.St{Gap: 2}, header, giode.Box(widgets.St{Padding: giode.Inset{Left: 12}}, widgets.Line(tail, theme.SizeFootnote, theme.T.Faint)))
	})
	if !open {
		return
	}
	st := widgets.St{FontSize: theme.SizeFootnote, Color: theme.T.Muted, LineHeight: 20}
	for j, chunk := range splitPlain(strings.TrimSpace(text), chunkSize) {
		rows := fmt.Sprintf("think%d/%d", idx, j)
		addText(rows, textToken(chunk)+a.textEnv(), func() widgets.El {
			return widgets.Row(widgets.St{Gap: 12, Align: giode.AlignStart, Padding: giode.Inset{Left: 12}, Width: -1},
				giode.Box(widgets.St{Width: 2, Height: -1, MinHeight: 16, Background: theme.T.Border}),
				giode.Box(widgets.St{FlexGrow: 1}, giode.Text(chunk, st)))
		})
	}
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
