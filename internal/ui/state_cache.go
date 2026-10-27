package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/giode/components/scroll"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// scroll returns the scroll container kept under key.
func (a *App) scroll(key string, content func() widgets.El) widgets.El {
	s, ok := a.scrolls[key]
	if !ok {
		holder := &struct{ fn func() widgets.El }{}
		s = giode.Scroll(func() widgets.El { return holder.fn() }, widgets.St{ScrollBar: scrollBar()})
		a.scrolls[key] = s
		scrollContent[s] = holder
	}
	scrollContent[s].fn = content
	return s
}

var scrollContent = map[*scroll.Scroll]*struct{ fn func() widgets.El }{}

// input returns the text field under key, created with init text.
func (a *App) input(key, placeholder, init string, opts ...func(*input.Input)) *input.Input {
	b, ok := a.inputs[key]
	if !ok {
		in := giode.Input.Text(placeholder, widgets.InputStyles())
		// Options first: a single-line field strips newlines from text.
		for _, o := range opts {
			o(in)
		}
		in.SetText(init)
		b = in
		a.inputs[key] = b
	}
	return b
}

func (a *App) dropInput(key string) { delete(a.inputs, key) }

// scrollBar styles scroll bars: thin on desktops, hidden on phones and
// tablets, where swiping needs no bar. Giode has no switch to hide one
// and draws its defaults for a fully transparent color, so a nearly
// transparent one stands in.
func scrollBar() giode.ScrollBar {
	if isMobile() {
		return giode.ScrollBar{Width: 4, ThumbColor: "#00000001", TrackColor: "#00000001"}
	}
	return giode.ScrollBar{Width: 4, ThumbColor: "theme.border", Radius: 2}
}
