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
	// composerH is the floating composer's height in dp, from the last
	// frame: the list leaves that much room below its last row.
	composerH float32
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
		Version(func(i int) any {
			last := i == len(v.rows)-1
			return fmt.Sprint(v.rows[i].Token, "|", v.rows[i].Gap, "|", v.bottomPad(last))
		}).
		Anchor(giode.LazyAnchorBottom)
	return v
}

func (v *chatView) view(compact bool) widgets.El {
	a := v.a
	if a.composer.full {
		return a.composer.fullView()
	}
	header := v.header(compact)
	if a.current == nil || len(a.current.Messages) == 0 {
		return widgets.Col(widgets.St{Width: -1, Height: -1},
			header,
			giode.Box(widgets.St{FlexGrow: 1, Width: -1, FlexDirection: giode.FlexDirectionColumn}, v.emptyState(compact)),
			a.composer.view(compact),
		)
	}
	// The composer floats over the messages, which stay visible around
	// its edges; the list leaves room under its last row to scroll it
	// clear.
	a.composer.floating = true
	composer := widgets.Measure(a.composer.view(compact), func(dp float32) { v.composerH = dp })
	a.composer.floating = false
	return widgets.Col(widgets.St{Width: -1, Height: -1},
		header,
		giode.Box(widgets.St{FlexGrow: 1, Width: -1},
			giode.Box(widgets.St{Position: giode.PositionAbsolute, Width: -1, Height: -1, FlexDirection: giode.FlexDirectionColumn}, v.listView()),
			giode.Box(widgets.St{Position: giode.PositionAbsolute, Width: -1, Bottom: composerGap}, composer)),
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
	pad := giode.Inset{Left: 16, Right: 16, Top: r.Gap, Bottom: v.bottomPad(i == len(v.rows)-1)}
	return giode.Box(widgets.St{Width: -1, Padding: pad}, widgets.Centered(780, r.Build()))
}

// bottomPad is the room under a row: under the last one, enough to
// scroll it above the floating composer.
func (v *chatView) bottomPad(last bool) int {
	if !last {
		return 0
	}
	return int(v.composerH) + composerGap + 16
}

// composerGap is the space under the floating composer, in dp.
const composerGap = 8
