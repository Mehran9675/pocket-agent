package markdown

import "testing"

func TestParseBlocks(t *testing.T) {
	var d Doc
	d.Set("# Title\n\nSome **bold** and [a link](https://x.dev).\n\n- [x] done\n- todo\n\n```go\nfmt.Println(1)\n```\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	kinds := []mdKind{mdHeading, mdPara, mdList, mdCode, mdTable}
	if len(d.blocks) != len(kinds) {
		t.Fatalf("got %d blocks: %+v", len(d.blocks), d.blocks)
	}
	for i, k := range kinds {
		if d.blocks[i].kind != k {
			t.Fatalf("block %d kind %v", i, d.blocks[i].kind)
		}
	}
	p := d.blocks[1].spans
	var bold, link bool
	for _, s := range p {
		bold = bold || (s.bold && s.text == "bold")
		link = link || (s.link == "https://x.dev" && s.text == "a link")
	}
	if !bold || !link {
		t.Fatalf("spans %+v", p)
	}
	if d.blocks[2].checks[0] != 1 || d.blocks[2].checks[1] != -1 {
		t.Fatalf("checks %v", d.blocks[2].checks)
	}
	if d.blocks[3].lang != "go" || d.blocks[3].code != "fmt.Println(1)" {
		t.Fatalf("code %+v", d.blocks[3])
	}
	if len(d.blocks[4].header) != 2 || len(d.blocks[4].rows) != 1 {
		t.Fatalf("table %+v", d.blocks[4])
	}
}
