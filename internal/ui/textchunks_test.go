package ui

import (
	"strings"
	"testing"
)

func TestSplitMarkdown(t *testing.T) {
	para := strings.Repeat("word ", 100) + "\n\n"
	code := "```go\n" + strings.Repeat("x := 1\n\n", 100) + "```\n\n"
	s := para + para + code + para + para
	chunks := splitMarkdown(s, 600)
	if strings.Join(chunks, "") != s {
		t.Fatal("chunks lose text")
	}
	if len(chunks) < 3 {
		t.Fatalf("got %d chunks", len(chunks))
	}
	for _, c := range chunks {
		if strings.Count(c, "```")%2 != 0 {
			t.Fatalf("chunk splits a code block: %q", c[:40])
		}
	}
	if got := splitMarkdown("short", 600); len(got) != 1 {
		t.Fatal("short text split")
	}
}

func TestSplitPlain(t *testing.T) {
	s := strings.Repeat("héllo wörld ", 500)
	chunks := splitPlain(s, 300)
	if len(chunks) < 10 {
		t.Fatalf("got %d chunks", len(chunks))
	}
	for _, c := range chunks {
		if len(c) > 300 {
			t.Fatalf("chunk of %d bytes", len(c))
		}
	}
	one := strings.Repeat("x", 1000)
	if strings.Join(splitPlain(one, 300), "") != one {
		t.Fatal("unbroken text lost")
	}
}
