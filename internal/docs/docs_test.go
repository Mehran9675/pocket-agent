package docs

import (
	"strings"
	"testing"
)

func TestHTMLText(t *testing.T) {
	page := `<html><head><title>T</title><script>x()</script></head><body><nav>menu</nav>
<main><h1>Hello</h1><p>One <b>bold</b> word.</p><ul><li>a</li><li>b</li></ul><pre>code
  here</pre></main><footer>f</footer></body></html>`
	title, text := HTMLText([]byte(page))
	if title != "T" {
		t.Errorf("title %q", title)
	}
	for _, want := range []string{"# Hello", "One bold word.", "- a", "- b", "code\n  here"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %q", want, text)
		}
	}
	for _, bad := range []string{"menu", "x()", "f\n"} {
		if strings.Contains(text, bad) {
			t.Errorf("unexpected %q in %q", bad, text)
		}
	}
}

func TestExtractText(t *testing.T) {
	r, err := Extract("notes.md", []byte("# Title\nbody"))
	if err != nil || r.Kind != KindDocument || !strings.Contains(r.Text, "body") {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := Extract("blob.bin", []byte{0, 1, 2, 0xff}); err == nil {
		t.Fatal("binary should be unsupported")
	}
}
