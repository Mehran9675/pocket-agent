package ui

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/local"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

const localesDir = "../../locales"

// englishOne gives the singular of each count phrase; the key is the
// plural.
var englishOne = map[string]string{
	"{n} tools":                      "{n} tool",
	"up to {n} steps":                "up to {n} step",
	"{n} pages":                      "{n} page",
	"{n} words":                      "{n} word",
	"{n} likes":                      "{n} like",
	"Connected · {n} tools":          "Connected · {n} tool",
	"{n} models · search to see all": "{n} model · search to see all",
	"Connected · {n} models":         "Connected · {n} model",
	"{n} sources":                    "{n} source",
	"{n} notes":                      "{n} note",
	"{n} read":                       "{n} read",
	"up to {n} tokens":               "up to {n} token",
	"{n} connected":                  "{n} connected",
	"{n} installed":                  "{n} installed",
	"{n} tools in chat":              "{n} tool in chat",
	"{n} configured":                 "{n} configured",
	"{n} agents":                     "{n} agent",
	"{n} memories":                   "{n} memory",
	"+{n} more":                      "+{n} more",
}

// uiKeys collects the text the UI translates: tr, trn and key calls in
// this package, and the built-in content other packages show.
func uiKeys(t *testing.T) (plain, plural []string) {
	t.Helper()
	seen := map[string]bool{}
	addPlain := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			plain = append(plain, s)
		}
	}
	files, _ := filepath.Glob("*.go")
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(af, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || (id.Name != "tr" && id.Name != "trn" && id.Name != "key") {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, _ := strconv.Unquote(lit.Value)
			if id.Name == "trn" {
				if !seen[s] {
					seen[s] = true
					plural = append(plural, s)
				}
				return true
			}
			addPlain(s)
			return true
		})
	}
	for _, ag := range store.BuiltinAgents() {
		addPlain(ag.Name)
		addPlain(ag.Description)
		for _, s := range ag.Starters {
			addPlain(s)
		}
	}
	for _, tl := range append(tools.Builtins(), tools.SkillTools()...) {
		addPlain(tl.Title)
		addPlain(tl.Group)
		addPlain(firstSentence(tl.Description))
	}
	for _, m := range local.Catalog {
		addPlain(m.Description)
	}
	for _, th := range []llm.Thinking{llm.ThinkingAuto, llm.ThinkingOff, llm.ThinkingLow, llm.ThinkingMedium, llm.ThinkingHigh} {
		addPlain(th.Label())
	}
	// agent.FriendlyError's own sentences.
	addPlain("Add an API key for this provider in Settings → Models.")
	addPlain("The request timed out.")
	sort.Strings(plain)
	sort.Strings(plural)
	return plain, plural
}

// TestEnglishMessages keeps locales/en.json - the list translators
// work from - in step with the code. UPDATE_LOCALES=1 rewrites it.
func TestEnglishMessages(t *testing.T) {
	plain, plural := uiKeys(t)
	en := map[string]any{}
	for _, k := range plain {
		en[k] = k
	}
	for _, k := range plural {
		one, ok := englishOne[k]
		if !ok {
			t.Errorf("no singular for count phrase %q in englishOne", k)
			one = k
		}
		en[k] = map[string]string{"one": one, "other": k}
	}
	want, _ := json.MarshalIndent(en, "", "  ")
	want = append(want, '\n')
	path := filepath.Join(localesDir, "en.json")
	if os.Getenv("UPDATE_LOCALES") != "" {
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(want) {
		t.Fatalf("locales/en.json is out of date; run UPDATE_LOCALES=1 go test ./internal/ui -run TestEnglishMessages")
	}
}

var placeholderRE = regexp.MustCompile(`\{[a-z]+\}|%[-+ #0]*[0-9]*(\.[0-9]+)?[a-z%]`)

func placeholders(s string) string {
	ps := placeholderRE.FindAllString(s, -1)
	sort.Strings(ps)
	return strings.Join(ps, " ")
}

// TestTranslations checks every locale translates every key, keeps
// its placeholders, and gives count phrases plural forms.
func TestTranslations(t *testing.T) {
	plain, plural := uiKeys(t)
	files, _ := filepath.Glob(filepath.Join(localesDir, "*.json"))
	if len(files) < 2 {
		t.Skip("no translations")
	}
	for _, f := range files {
		code := strings.TrimSuffix(filepath.Base(f), ".json")
		if code == "en" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var msgs map[string]any
		if err := json.Unmarshal(b, &msgs); err != nil {
			t.Errorf("%s: %v", code, err)
			continue
		}
		missing := 0
		for _, k := range plain {
			v, ok := msgs[k].(string)
			if !ok || strings.TrimSpace(v) == "" {
				missing++
				if missing <= 5 {
					t.Errorf("%s: missing %q", code, k)
				}
				continue
			}
			if placeholders(v) != placeholders(k) {
				t.Errorf("%s: %q keeps %q, want %q", code, k, placeholders(v), placeholders(k))
			}
		}
		for _, k := range plural {
			forms, ok := msgs[k].(map[string]any)
			if !ok || forms["other"] == nil {
				missing++
				if missing <= 5 {
					t.Errorf("%s: missing plural forms for %q", code, k)
				}
				continue
			}
			for form, v := range forms {
				s, _ := v.(string)
				if !strings.Contains(s, "{n}") && placeholders(k) == "{n}" && form != "one" && form != "zero" {
					t.Errorf("%s: %q form %s lacks {n}", code, k, form)
				}
			}
		}
		if missing > 5 {
			t.Errorf("%s: %d keys missing in all", code, missing)
		}
	}
}
