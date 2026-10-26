package docs

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTMLText extracts a page's title and readable text as light
// Markdown: headings, paragraphs, list items, code blocks and table
// rows, with navigation, scripts, styles and forms dropped.
func HTMLText(data []byte) (title, text string) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return "", string(data)
	}
	root := findMain(doc)
	var b strings.Builder
	var walk func(n *html.Node, pre bool)
	walk = func(n *html.Node, pre bool) {
		if n.Type == html.ElementNode {
			switch n.DataAtom {
			case atom.Script, atom.Style, atom.Noscript, atom.Svg, atom.Nav, atom.Footer,
				atom.Form, atom.Button, atom.Iframe, atom.Template, atom.Select, atom.Aside:
				return
			case atom.Header:
				if n != root {
					return
				}
			}
			if hidden(n) {
				return
			}
		}
		switch n.Type {
		case html.TextNode:
			if pre {
				b.WriteString(n.Data)
				return
			}
			s := strings.Join(strings.Fields(n.Data), " ")
			if s == "" {
				if strings.ContainsAny(n.Data, " \n\t") && b.Len() > 0 && !strings.HasSuffix(b.String(), " ") && !strings.HasSuffix(b.String(), "\n") {
					b.WriteByte(' ')
				}
				return
			}
			if (n.Data[0] == ' ' || n.Data[0] == '\n') && b.Len() > 0 && !strings.HasSuffix(b.String(), " ") && !strings.HasSuffix(b.String(), "\n") {
				b.WriteByte(' ')
			}
			b.WriteString(s)
			if last := n.Data[len(n.Data)-1]; last == ' ' || last == '\n' {
				b.WriteByte(' ')
			}
			return
		case html.ElementNode:
			block := ""
			switch n.DataAtom {
			case atom.H1:
				block = "# "
			case atom.H2:
				block = "## "
			case atom.H3:
				block = "### "
			case atom.H4, atom.H5, atom.H6:
				block = "#### "
			case atom.Li:
				block = "- "
			case atom.P, atom.Div, atom.Section, atom.Article, atom.Blockquote, atom.Tr, atom.Dt, atom.Dd, atom.Figcaption, atom.Table:
				block = "\n"
			case atom.Br:
				b.WriteByte('\n')
				return
			case atom.Pre:
				b.WriteString("\n```\n")
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c, true)
				}
				b.WriteString("\n```\n")
				return
			case atom.Td, atom.Th:
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c, pre)
				}
				b.WriteString(" | ")
				return
			case atom.Img:
				if alt := attr(n, "alt"); alt != "" {
					b.WriteString("[image: " + alt + "]")
				}
				return
			}
			if block != "" {
				ensureNewline(&b)
				if block != "\n" {
					b.WriteString(block)
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c, pre)
			}
			if block != "" {
				ensureNewline(&b)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, pre)
		}
	}
	walk(root, false)
	return findTitle(doc), collapseBlankLines(b.String())
}

func ensureNewline(b *strings.Builder) {
	s := b.String()
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		b.WriteByte('\n')
	}
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hidden(n *html.Node) bool {
	if attr(n, "aria-hidden") == "true" || attr(n, "hidden") != "" {
		return true
	}
	st := strings.ReplaceAll(attr(n, "style"), " ", "")
	if strings.Contains(st, "display:none") {
		return true
	}
	role := attr(n, "role")
	return role == "navigation" || role == "banner" || role == "contentinfo"
}

// findMain returns <main> or <article> when the page has one, else
// <body>.
func findMain(doc *html.Node) *html.Node {
	var mainEl, article, body *html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch {
			case n.DataAtom == atom.Main && mainEl == nil:
				mainEl = n
			case n.DataAtom == atom.Article && article == nil:
				article = n
			case n.DataAtom == atom.Body && body == nil:
				body = n
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	switch {
	case mainEl != nil:
		return mainEl
	case article != nil:
		return article
	case body != nil:
		return body
	}
	return doc
}

func findTitle(doc *html.Node) string {
	var title string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if title != "" {
			return
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.Title && n.FirstChild != nil {
			title = strings.TrimSpace(n.FirstChild.Data)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return title
}
