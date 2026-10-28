package skills

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

const pdfSkill = `---
name: pdf-tools
description: >
  Fill and merge PDF forms.
  Use when the user mentions PDFs.
license: MIT
---

# PDF tools

See reference/forms.md.
`

func TestParseFoldedDescription(t *testing.T) {
	meta, body := parse(pdfSkill)
	if meta["name"] != "pdf-tools" || meta["description"] != "Fill and merge PDF forms. Use when the user mentions PDFs." {
		t.Fatalf("meta = %q", meta)
	}
	if !strings.HasPrefix(body, "\n# PDF tools") {
		t.Fatalf("body = %q", body)
	}
}

func TestComposeRoundTrips(t *testing.T) {
	src := Compose("notes", `Says "hi": always`, "Do it.")
	meta, body := parse(src)
	if meta["description"] != `Says "hi": always` || strings.TrimSpace(body) != "Do it." {
		t.Fatalf("meta=%q body=%q", meta, body)
	}
}

func TestInstallZipInFolder(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, text := range map[string]string{
		"pdf-tools/SKILL.md":            pdfSkill,
		"pdf-tools/reference/forms.md":  "Forms.",
		"__MACOSX/pdf-tools/._SKILL.md": "junk",
		"pdf-tools/examples/SKILL.md":   "---\nname: nested\n---\n",
	} {
		w, _ := zw.Create(name)
		w.Write([]byte(text))
	}
	zw.Close()
	l := Open(t.TempDir())
	s, err := l.Install("pdf.zip", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "pdf-tools" || strings.Join(s.Files, ",") != "examples/SKILL.md,reference/forms.md" {
		t.Fatalf("skill = %+v", s)
	}
	if text, err := l.ReadFile("pdf-tools", "../../reference/forms.md"); err != nil || text != "Forms." {
		t.Fatalf("read = %q, %v", text, err)
	}
}

func TestInstallRejectsMissingDescription(t *testing.T) {
	if _, err := Open(t.TempDir()).Install("SKILL.md", []byte("---\nname: x\n---\nBody")); err == nil {
		t.Fatal("want an error")
	}
}

func TestSaveRenames(t *testing.T) {
	l := Open(t.TempDir())
	if _, err := l.Create("My Notes", "Take notes.", "Write them down."); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Save("my-notes", Compose("notes", "Take notes.", "Write.")); err != nil {
		t.Fatal(err)
	}
	if got := l.List(); len(got) != 1 || got[0].Name != "notes" {
		t.Fatalf("list = %+v", got)
	}
}

func TestRawURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://github.com/a/b/blob/main/x/SKILL.md": "https://raw.githubusercontent.com/a/b/main/x/SKILL.md",
		"https://github.com/a/b/tree/main/skills/pdf": "https://raw.githubusercontent.com/a/b/main/skills/pdf/SKILL.md",
		"https://example.com/s.zip":                   "https://example.com/s.zip",
	} {
		if got := rawURL(in); got != want {
			t.Errorf("rawURL(%s) = %s", in, got)
		}
	}
}

func TestGithubTree(t *testing.T) {
	o, r, ref, dir, ok := githubTree("https://github.com/anthropics/skills/tree/main/skills/pdf/")
	if !ok || o != "anthropics" || r != "skills" || ref != "main" || dir != "skills/pdf" {
		t.Fatalf("%s %s %s %s %v", o, r, ref, dir, ok)
	}
	if _, _, _, _, ok := githubTree("https://github.com/a/b/blob/main/SKILL.md"); ok {
		t.Fatal("blob link taken as a folder")
	}
}

// fakeGitHub serves a skill folder the way GitHub's contents API does.
type fakeGitHub map[string]string

func (f fakeGitHub) RoundTrip(r *http.Request) (*http.Response, error) {
	body, ok := f[r.URL.String()]
	code := http.StatusOK
	if !ok {
		code = http.StatusNotFound
	}
	return &http.Response{StatusCode: code, Status: http.StatusText(code), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func TestInstallGithubFolder(t *testing.T) {
	const api = "https://api.github.com/repos/o/r/contents/skills/pdf"
	old := http.DefaultClient.Transport
	http.DefaultClient.Transport = fakeGitHub{
		api + "?ref=main": `[{"name":"SKILL.md","type":"file","size":10,"download_url":"https://raw/SKILL.md"},
			{"name":"ref","type":"dir"}]`,
		api + "/ref?ref=main":  `[{"name":"forms.md","type":"file","size":6,"download_url":"https://raw/forms.md"}]`,
		"https://raw/SKILL.md": pdfSkill,
		"https://raw/forms.md": "Forms.",
	}
	defer func() { http.DefaultClient.Transport = old }()
	l := Open(t.TempDir())
	s, err := l.InstallURL(context.Background(), "https://github.com/o/r/tree/main/skills/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "pdf-tools" || strings.Join(s.Files, ",") != "ref/forms.md" {
		t.Fatalf("skill = %+v", s)
	}
}
