package markdown

// span is an inline run of styled text.
type span struct {
	text                       string
	bold, italic, code, strike bool
	link                       string
}
