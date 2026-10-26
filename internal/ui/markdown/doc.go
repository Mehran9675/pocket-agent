// Package markdown renders Markdown (CommonMark + GFM tables, task lists,
// strikethrough) to Giode elements, with rich inline text and tappable
// links, code blocks and tables.
package markdown

import (
	"strconv"

	"gioui.org/x/richtext"
	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
	"github.com/yuin/goldmark/text"
)

// Doc is a parsed Markdown document with the interactive state
// of its links. Keep one per text you render; Set re-parses only when
// the text changes.
type Doc struct {
	src    string
	blocks []mdBlock
	states map[string]*richtext.InteractiveText
	// OnLink handles a tapped link.
	OnLink func(url string)
	// OnCopy copies a code block.
	OnCopy func(code string)
	// Key namespaces the doc's keyed widgets.
	Key string
}

// Set updates the source text.
func (d *Doc) Set(src string) {
	if src == d.src && d.blocks != nil {
		return
	}
	d.src = src
	b := []byte(src)
	root := mdParser.Parser().Parse(text.NewReader(b))
	d.blocks = convertBlocks(root, b)
}

// Element renders the document.
func (d *Doc) Element(base float32) widgets.El {
	if d.states == nil {
		d.states = map[string]*richtext.InteractiveText{}
	}
	return widgets.Col(widgets.St{Gap: 12, Width: -1}, d.blocksEls(d.blocks, base, "b")...)
}

func (d *Doc) blocksEls(blocks []mdBlock, base float32, path string) []widgets.El {
	var out []widgets.El
	for i, b := range blocks {
		key := path + "." + strconv.Itoa(i)
		out = append(out, d.blockEl(b, base, key))
	}
	return out
}

func (d *Doc) blockEl(b mdBlock, base float32, key string) widgets.El {
	switch b.kind {
	case mdHeading:
		size := base
		top := 6
		switch b.level {
		case 1:
			size, top = base+6, 10
		case 2:
			size, top = base+4, 8
		case 3:
			size = base + 2
		}
		return giode.Box(widgets.St{Padding: giode.Inset{Top: top}}, d.rich(key, b.spans, size, theme.T.Text, true))
	case mdRule:
		return giode.Box(widgets.St{Padding: giode.SymmetricInset(0, 6), Width: -1}, widgets.HDivider())
	case mdCode:
		return d.codeBlock(key, b)
	case mdQuote:
		return widgets.Row(widgets.St{Gap: 12, Align: giode.AlignStart},
			giode.Box(widgets.St{Width: 3, Height: -1, MinHeight: 20, BorderRadius: 2, Background: theme.T.BorderStrong}),
			widgets.Col(widgets.St{Gap: 10, FlexGrow: 1}, d.blocksEls(b.quote, base, key)...))
	case mdList:
		var items []widgets.El
		for i, it := range b.items {
			marker := "•"
			markerColor := theme.T.Muted
			if b.ordered {
				marker = strconv.Itoa(b.start+i) + "."
			}
			var markerEl widgets.El = giode.Box(widgets.St{Width: 22, Padding: giode.Inset{Top: 1}}, widgets.Text(marker, base, markerColor, pr.FontWeightMedium))
			if b.checks[i] >= 0 {
				name, c := "circle", theme.T.Muted
				if b.checks[i] == 1 {
					name, c = "circle-check", theme.T.Success
				}
				markerEl = giode.Box(widgets.St{Width: 24, Padding: giode.Inset{Top: 2}}, widgets.Icon(name, 17, c))
			}
			items = append(items, widgets.Row(widgets.St{Align: giode.AlignStart, Width: -1},
				markerEl,
				widgets.Col(widgets.St{Gap: 8, FlexGrow: 1}, d.blocksEls(it, base, key+".i"+strconv.Itoa(i))...)))
		}
		return widgets.Col(widgets.St{Gap: 6, Width: -1}, items...)
	case mdTable:
		return d.table(key, b, base)
	}
	return d.rich(key, b.spans, base, theme.T.Text, false)
}
