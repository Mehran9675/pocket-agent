package ui

import "strings"

// chunkSize is the length long text is split at for rendering. Each
// chunk is its own memo, so touching or streaming into one part of a
// long reply re-lays out only that part: giode rebuilds a memo on every
// frame the pointer is over it.
const chunkSize = 2000

// splitMarkdown splits Markdown into chunks of about max bytes at blank
// lines outside fenced code blocks, so each chunk renders on its own as
// the whole would. Text with no such break stays one chunk.
func splitMarkdown(s string, max int) []string {
	if len(s) <= max {
		return []string{s}
	}
	var chunks []string
	start, fence := 0, ""
	lines := strings.SplitAfter(s, "\n")
	pos := 0
	for i, line := range lines {
		pos += len(line)
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			switch {
			case fence == "":
				fence = t[:3]
			case strings.HasPrefix(t, fence):
				fence = ""
			}
		}
		// Break before a blank line once the chunk is long enough.
		if fence == "" && pos-start >= max && t == "" && i+1 < len(lines) {
			chunks = append(chunks, s[start:pos])
			start = pos
		}
	}
	if start < len(s) {
		chunks = append(chunks, s[start:])
	}
	return chunks
}

// splitPlain splits plain text into chunks of about max bytes at line
// breaks, or at spaces in a very long line.
func splitPlain(s string, max int) []string {
	var chunks []string
	for len(s) > max {
		cut := strings.LastIndexByte(s[:max], '\n')
		if cut < max/2 {
			cut = strings.LastIndexByte(s[:max], ' ')
		}
		if cut < max/2 {
			cut = max
			for cut < len(s) && s[cut]&0xC0 == 0x80 { // stay on a rune boundary
				cut++
			}
		}
		chunks = append(chunks, s[:cut])
		s = strings.TrimPrefix(s[cut:], "\n")
	}
	return append(chunks, s)
}
