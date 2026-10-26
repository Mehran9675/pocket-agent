package markdown

import (
	east "github.com/yuin/goldmark/extension/ast"
)

type mdKind int

const (
	mdPara mdKind = iota
	mdHeading
	mdList
	mdCode
	mdQuote
	mdRule
	mdTable
)

// mdBlock is a block of a parsed document.
type mdBlock struct {
	kind    mdKind
	level   int // heading level
	spans   []span
	ordered bool
	start   int
	items   [][]mdBlock
	checks  []int // per item: -1 none, 0 unchecked, 1 checked
	lang    string
	code    string
	quote   []mdBlock
	header  [][]span
	rows    [][][]span
	align   []east.Alignment
}
