package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

type chatView struct {
	a *App
	// list builds only the rows on screen; rows are this frame's.
	list *giode.LazyListElement
	rows []msgRow
	// jumpToBottom scrolls to the newest message on the next frame.
	jumpToBottom bool
}

func newChatView(a *App) *chatView {
	v := &chatView{a: a}
	// Anchored to the bottom: follows a streaming reply until the user
	// scrolls up, and resumes when they scroll back down.
	v.list = giode.LazyList(
		func() int { return len(v.rows) },
		func(i int) any { return v.rows[i].Key },
		func(i int) giode.Element { return v.row(i) },
		widgets.St{ScrollBar: scrollBar()},
	).
		Version(func(i int) any { return fmt.Sprint(v.rows[i].Token, "|", v.rows[i].Gap, "|", i == len(v.rows)-1) }).
		Anchor(giode.LazyAnchorBottom)
	return v
}

func (v *chatView) view(compact bool) widgets.El {
	a := v.a
	if a.composer.full {
		return a.composer.fullView()
	}
	header := v.header(compact)
	var body widgets.El
	if a.current == nil || len(a.current.Messages) == 0 {
		body = v.emptyState(compact)
	} else {
		body = v.listView()
	}
	return widgets.Col(widgets.St{Width: -1, Height: -1},
		header,
		giode.Box(widgets.St{FlexGrow: 1, Width: -1, FlexDirection: giode.FlexDirectionColumn}, body),
		a.composer.view(compact),
	)
}

// listView lays out the message list. Text across messages selects as
// one, like a web page.
func (v *chatView) listView() widgets.El {
	v.rows = v.messageRows()
	if v.jumpToBottom {
		v.jumpToBottom = false
		v.list.ScrollToBottom()
	}
	return giode.SelectionArea("chat/"+v.a.current.ID, giode.Box(widgets.St{Width: -1, FlexGrow: 1, FlexDirection: giode.FlexDirectionColumn}, v.list))
}

// messageRows flattens the conversation into list rows.
func (v *chatView) messageRows() []msgRow {
	a := v.a
	c := a.current
	var rows []msgRow
	for i, m := range c.Messages {
		mr := a.messageRows(c, m, i == len(c.Messages)-1)
		if len(mr) > 0 {
			mr[0].Gap = 22 // between messages
			if i == 0 {
				mr[0].Gap = 8
			}
		}
		rows = append(rows, mr...)
	}
	return rows
}

// row lays out row i in the reading column.
func (v *chatView) row(i int) giode.Element {
	r := v.rows[i]
	pad := giode.Inset{Left: 16, Right: 16, Top: r.Gap}
	if i == len(v.rows)-1 {
		pad.Bottom = 24
	}
	return giode.Box(widgets.St{Width: -1, Padding: pad}, widgets.Centered(780, r.Build()))
}
