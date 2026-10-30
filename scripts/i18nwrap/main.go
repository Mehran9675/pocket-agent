// Command i18nwrap finds user-visible string literals in Go files and,
// with -w, wraps them in tr(...). It is a helper for translating the
// UI: review its report before writing.
//
//	go run ./scripts/i18nwrap internal/ui/*.go        # report
//	go run ./scripts/i18nwrap -w internal/ui/*.go     # rewrite
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var keyLike = regexp.MustCompile(`^[a-z0-9_.\-#:%/+*?=&|<>()\[\]{}@!,]*$`)

// visible reports whether s reads as text for people.
func visible(s string) bool {
	if strings.TrimSpace(s) == "" || !strings.ContainsFunc(s, func(r rune) bool { return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' }) {
		return false
	}
	if !strings.Contains(s, " ") && (keyLike.MatchString(s) || strings.Contains(s, "/") || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "theme.")) {
		return false
	}
	if strings.HasPrefix(s, "http") || strings.HasPrefix(s, "%") && !strings.Contains(s, " ") {
		return false
	}
	return true
}

// skipCalls never take UI text.
var skipCalls = map[string]bool{
	"Errorf": true, "New": true, "Print": true, "Printf": true, "Println": true, "Fatal": true, "Fatalf": true,
	"HasPrefix": true, "HasSuffix": true, "Contains": true, "TrimPrefix": true, "TrimSuffix": true, "Split": true,
	"Index": true, "Replace": true, "ReplaceAll": true, "EqualFold": true, "Cut": true, "Join": true, "Fields": true,
	"tr": true, "trn": true, "Getenv": true, "MustCompile": true, "Complete": true, "Fprintf": true, "Fprintln": true,
	"WriteString": true, "Fprint": true, "Debug": true, "Info": true, "Warn": true, "Error": true,
}

type hit struct {
	pos, end int
	line     int
	text     string
	note     string
}

func main() {
	write := flag.Bool("w", false, "rewrite files")
	flag.Parse()
	total := 0
	for _, path := range flag.Args() {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var hits []hit
		var stack []ast.Node
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			stack = append(stack, n)
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil || !visible(s) {
				return true
			}
			note := ""
			for i := len(stack) - 2; i >= 0; i-- {
				switch p := stack[i].(type) {
				case *ast.ImportSpec, *ast.Field:
					return true
				case *ast.CaseClause:
					for _, e := range p.List {
						if e == stack[i+1] {
							return true
						}
					}
					i = -1
				case *ast.BinaryExpr:
					if p.Op == token.EQL || p.Op == token.NEQ {
						return true
					}
				case *ast.CallExpr:
					name := ""
					switch fn := p.Fun.(type) {
					case *ast.Ident:
						name = fn.Name
					case *ast.SelectorExpr:
						name = fn.Sel.Name
					}
					if skipCalls[name] && i == len(stack)-2 {
						return true
					}
				case *ast.GenDecl:
					if p.Tok == token.CONST || p.Tok == token.VAR {
						note = "package-level " + p.Tok.String()
					}
				case *ast.FuncDecl:
					i = -1
				}
			}
			hits = append(hits, hit{pos: fset.Position(lit.Pos()).Offset, end: fset.Position(lit.End()).Offset, line: fset.Position(lit.Pos()).Line, text: s, note: note})
			return true
		})
		for _, h := range hits {
			total++
			fmt.Printf("%s:%d\t%q", path, h.line, h.text)
			if h.note != "" {
				fmt.Printf("\t[%s]", h.note)
			}
			fmt.Println()
		}
		if *write && len(hits) > 0 {
			sort.Slice(hits, func(i, j int) bool { return hits[i].pos > hits[j].pos })
			out := src
			for _, h := range hits {
				if h.note != "" {
					continue
				}
				out = append(out[:h.pos:h.pos], append([]byte("tr("+string(src[h.pos:h.end])+")"), out[h.end:]...)...)
			}
			if err := os.WriteFile(path, out, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
	fmt.Fprintln(os.Stderr, total, "strings")
}
