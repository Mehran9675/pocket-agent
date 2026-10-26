package llm

import "strings"

// ThinkSplitter routes text inside <think>...</think> tags - how
// open-weight reasoning models (Qwen3, DeepSeek-R1, ...) emit their
// thinking - to reasoning instead of the reply. It copes with tags
// split across stream chunks.
type ThinkSplitter struct {
	inThink bool
	pending string
	started bool
}

const (
	thinkOpen  = "<think>"
	thinkClose = "</think>"
)

// Feed consumes a chunk and returns the reply and reasoning text it
// completes.
func (s *ThinkSplitter) Feed(chunk string) (text, reasoning string) {
	buf := s.pending + chunk
	s.pending = ""
	var t, r strings.Builder
	for len(buf) > 0 {
		tag := thinkOpen
		if s.inThink {
			tag = thinkClose
		}
		i := strings.Index(buf, tag)
		if i < 0 {
			// Keep a possible partial tag at the end for the next chunk.
			keep := partialSuffix(buf, tag)
			emit := buf[:len(buf)-keep]
			s.pending = buf[len(buf)-keep:]
			if s.inThink {
				r.WriteString(emit)
			} else {
				t.WriteString(emit)
			}
			break
		}
		if s.inThink {
			r.WriteString(buf[:i])
		} else {
			t.WriteString(buf[:i])
		}
		buf = buf[i+len(tag):]
		s.inThink = !s.inThink
		if !s.inThink {
			// Models put blank lines after </think>.
			buf = strings.TrimLeft(buf, "\n")
		}
	}
	out := t.String()
	if !s.started {
		// Drop leading whitespace before the reply proper starts.
		out = strings.TrimLeft(out, " \n")
		if out != "" {
			s.started = true
		}
	}
	return out, r.String()
}

// Flush returns whatever partial-tag text was held back.
func (s *ThinkSplitter) Flush() (text, reasoning string) {
	p := s.pending
	s.pending = ""
	if s.inThink {
		return "", p
	}
	return p, ""
}

// partialSuffix returns the length of the longest suffix of s that is
// a proper prefix of tag.
func partialSuffix(s, tag string) int {
	for n := min(len(tag)-1, len(s)); n > 0; n-- {
		if strings.HasSuffix(s, tag[:n]) {
			return n
		}
	}
	return 0
}
