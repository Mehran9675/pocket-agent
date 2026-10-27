package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/scroll"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

type chatView struct {
	a    *App
	list *scroll.Scroll
	// jumpToBottom scrolls to the newest message on the next frame.
	jumpToBottom bool
}

func newChatView(a *App) *chatView {
	v := &chatView{a: a}
	// Anchored to the bottom: follows a streaming reply until the user
	// scrolls up, and resumes when they scroll back down.
	v.list = giode.Scroll(func() widgets.El { return v.messages() },
		widgets.St{ScrollBar: giode.ScrollBar{Width: 4, ThumbColor: "theme.border", Radius: 2}}).
		Anchor(giode.AnchorBottom)
	return v
}

func (v *chatView) view(compact bool) widgets.El {
	a := v.a
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

// listView lays out the message scroll.
func (v *chatView) listView() widgets.El {
	if v.jumpToBottom {
		v.jumpToBottom = false
		v.list.ScrollToBottom()
	}
	return v.list
}

func (v *chatView) messages() widgets.El {
	a := v.a
	c := a.current
	if c == nil {
		return giode.Box(widgets.St{})
	}
	var kids []widgets.El
	for i, m := range c.Messages {
		last := i == len(c.Messages)-1
		kids = append(kids, a.messageView(c, m, last))
	}
	return giode.Box(widgets.St{Width: -1, Padding: giode.Inset{Left: 16, Right: 16, Top: 8, Bottom: 24}},
		widgets.Centered(780, widgets.Col(widgets.St{Width: -1, Gap: 22}, kids...)))
}
