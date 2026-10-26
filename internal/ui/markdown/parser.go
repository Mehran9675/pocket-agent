package markdown

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
)

var mdParser = goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.Linkify, extension.TaskList))

func convertBlocks(parent ast.Node, src []byte) []mdBlock {
	var out []mdBlock
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		switch n := n.(type) {
		case *ast.Paragraph, *ast.TextBlock:
			out = append(out, mdBlock{kind: mdPara, spans: inlines(n, src, span{})})
		case *ast.Heading:
			out = append(out, mdBlock{kind: mdHeading, level: n.Level, spans: inlines(n, src, span{bold: true})})
		case *ast.ThematicBreak:
			out = append(out, mdBlock{kind: mdRule})
		case *ast.FencedCodeBlock:
			out = append(out, mdBlock{kind: mdCode, lang: string(n.Language(src)), code: blockLines(n, src)})
		case *ast.CodeBlock:
			out = append(out, mdBlock{kind: mdCode, code: blockLines(n, src)})
		case *ast.Blockquote:
			out = append(out, mdBlock{kind: mdQuote, quote: convertBlocks(n, src)})
		case *ast.List:
			b := mdBlock{kind: mdList, ordered: n.IsOrdered(), start: n.Start}
			for it := n.FirstChild(); it != nil; it = it.NextSibling() {
				check := -1
				if tb := it.FirstChild(); tb != nil {
					if cb, ok := tb.FirstChild().(*east.TaskCheckBox); ok {
						check = 0
						if cb.IsChecked {
							check = 1
						}
					}
				}
				b.items = append(b.items, convertBlocks(it, src))
				b.checks = append(b.checks, check)
			}
			out = append(out, b)
		case *east.Table:
			b := mdBlock{kind: mdTable, align: n.Alignments}
			for r := n.FirstChild(); r != nil; r = r.NextSibling() {
				var cells [][]span
				for c := r.FirstChild(); c != nil; c = c.NextSibling() {
					cells = append(cells, inlines(c, src, span{}))
				}
				if _, ok := r.(*east.TableHeader); ok {
					b.header = cells
				} else {
					b.rows = append(b.rows, cells)
				}
			}
			out = append(out, b)
		case *ast.HTMLBlock:
			out = append(out, mdBlock{kind: mdPara, spans: []span{{text: strings.TrimSpace(blockLines(n, src))}}})
		default:
			if n.HasChildren() {
				out = append(out, convertBlocks(n, src)...)
			}
		}
	}
	return out
}

func blockLines(n ast.Node, src []byte) string {
	var b bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(src))
	}
	return strings.TrimRight(b.String(), "\n")
}

func inlines(parent ast.Node, src []byte, style span) []span {
	var out []span
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		switch n := n.(type) {
		case *ast.Text:
			s := style
			s.text = string(n.Segment.Value(src))
			if n.SoftLineBreak() {
				s.text += " "
			}
			if n.HardLineBreak() {
				s.text += "\n"
			}
			out = append(out, s)
		case *ast.String:
			s := style
			s.text = string(n.Value)
			out = append(out, s)
		case *ast.CodeSpan:
			s := style
			s.code = true
			var b strings.Builder
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					b.Write(t.Segment.Value(src))
				}
			}
			s.text = b.String()
			out = append(out, s)
		case *ast.Emphasis:
			s := style
			if n.Level >= 2 {
				s.bold = true
			} else {
				s.italic = true
			}
			out = append(out, inlines(n, src, s)...)
		case *east.Strikethrough:
			s := style
			s.strike = true
			out = append(out, inlines(n, src, s)...)
		case *ast.Link:
			s := style
			s.link = string(n.Destination)
			out = append(out, inlines(n, src, s)...)
		case *ast.AutoLink:
			s := style
			s.link = string(n.URL(src))
			s.text = string(n.Label(src))
			out = append(out, s)
		case *ast.Image:
			s := style
			s.link = string(n.Destination)
			alt := string(n.Text(src)) //nolint:staticcheck
			if alt == "" {
				alt = "image"
			}
			s.text = "[" + alt + "]"
			out = append(out, s)
		case *east.TaskCheckBox:
			// rendered by the list
		case *ast.RawHTML:
			// dropped
		default:
			out = append(out, inlines(n, src, style)...)
		}
	}
	return out
}

func spansText(sp []span) string {
	var b strings.Builder
	for _, s := range sp {
		b.WriteString(s.text)
	}
	return b.String()
}
