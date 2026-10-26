package markdown

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/x/richtext"
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/fonts"
	pr "github.com/mehran9675/giode/styles/properties"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// rich lays out inline spans with x/richtext, handling link taps.
func (d *Doc) rich(key string, sp []span, size float32, fg string, bold bool) widgets.El {
	state, ok := d.states[key]
	if !ok {
		state = &richtext.InteractiveText{}
		d.states[key] = state
	}
	fgc := pr.ResolveColor(fg)
	linkc := pr.ResolveColor(theme.T.PrimaryText)
	codec := pr.ResolveColor(theme.T.Text)
	if theme.T.Dark {
		codec = pr.ResolveColor("#e6c07b")
	} else {
		codec = pr.ResolveColor("#b4235a")
	}
	styles := make([]richtext.SpanStyle, 0, len(sp))
	for _, s := range sp {
		if s.text == "" {
			continue
		}
		f := font.Font{Typeface: theme.FontUI}
		if s.bold || bold {
			f.Weight = font.Bold
		}
		if s.italic {
			f.Style = font.Italic
		}
		c := fgc
		sz := unit.Sp(size)
		if s.code {
			f = font.Font{Typeface: theme.FontMono}
			c = codec
			sz = unit.Sp(size * 0.9)
		}
		if s.strike {
			c.A = c.A / 2
		}
		ss := richtext.SpanStyle{Font: f, Size: sz, Color: c, Content: s.text}
		if s.link != "" {
			ss.Color = linkc
			ss.Interactive = true
			ss.Set("url", s.link)
		}
		styles = append(styles, ss)
	}
	if len(styles) == 0 {
		return giode.Box(widgets.St{})
	}
	lineHeight := unit.Sp(size * 1.5)
	return giode.Raw(func(gtx layout.Context) layout.Dimensions {
		for {
			sp, ev, ok := state.Update(gtx)
			if !ok {
				break
			}
			if ev.Type == richtext.Click && sp != nil && d.OnLink != nil {
				if u, ok := sp.Get("url").(string); ok {
					d.OnLink(u)
				}
			}
		}
		ts := richtext.Text(state, fonts.Shaper(), styles...)
		ts.LineHeight = lineHeight
		gtx.Constraints.Min.X = 0
		return ts.Layout(gtx)
	})
}
