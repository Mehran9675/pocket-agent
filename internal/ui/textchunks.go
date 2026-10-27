package ui

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// chunkSize is the length long text is split at for rendering. Each
// chunk is its own memo, so touching or streaming into one part of a
// long reply re-lays out only that part: giode rebuilds a memo on every
// frame the pointer is over it.
const chunkSize = 2000

// splitMarkdown splits Markdown into chunks of about max bytes at blank
// lines outside fenced code blocks, so each chunk renders on its own as
// the whole would. Text with no blank line for 3*max bytes is split at
// a line break instead (a paragraph that long gains a break).
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
		// Break at a blank line once the chunk is long enough, or at any
		// line in text that has no blank lines for far too long.
		if fence == "" && i+1 < len(lines) && ((pos-start >= max && t == "") || pos-start >= 3*max) {
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

// textToken identifies a chunk's content for row height caching.
func textToken(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return strconv.FormatUint(h.Sum64(), 36)
}

// textEnv changes with the settings that change how text lays out.
func (a *App) textEnv() string {
	return fmt.Sprintf("|%g", a.settings.FontScale)
}

// textSize is the body text size for replies.
func (a *App) textSize() float32 {
	scale := a.settings.FontScale
	if scale == 0 {
		scale = 1
	}
	return float32(theme.SizeBody * scale)
}
