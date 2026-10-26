package tools

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Evaluate computes an arithmetic expression: + - * / % ^, parentheses,
// constants (pi, e) and functions (sqrt, abs, ln, log, log2, exp, sin,
// cos, tan, asin, acos, atan, floor, ceil, round, min, max, pow).
func Evaluate(expr string) (float64, error) {
	p := &calcParser{src: strings.ReplaceAll(strings.ReplaceAll(expr, "×", "*"), "÷", "/")}
	p.next()
	v, err := p.expr()
	if err != nil {
		return 0, err
	}
	if p.tok.kind != tokEOF {
		return 0, fmt.Errorf("unexpected %q", p.tok.text)
	}
	return v, nil
}

type tokKind int

const (
	tokEOF tokKind = iota
	tokNum
	tokIdent
	tokOp
)

type token struct {
	kind tokKind
	text string
	num  float64
}

type calcParser struct {
	src string
	pos int
	tok token
}

func (p *calcParser) next() {
	for p.pos < len(p.src) && unicode.IsSpace(rune(p.src[p.pos])) {
		p.pos++
	}
	if p.pos >= len(p.src) {
		p.tok = token{kind: tokEOF}
		return
	}
	c := p.src[p.pos]
	switch {
	case c >= '0' && c <= '9' || c == '.':
		start := p.pos
		for p.pos < len(p.src) && (p.src[p.pos] >= '0' && p.src[p.pos] <= '9' || p.src[p.pos] == '.' || p.src[p.pos] == '_') {
			p.pos++
		}
		if p.pos < len(p.src) && (p.src[p.pos] == 'e' || p.src[p.pos] == 'E') {
			q := p.pos + 1
			if q < len(p.src) && (p.src[q] == '+' || p.src[q] == '-') {
				q++
			}
			if q < len(p.src) && p.src[q] >= '0' && p.src[q] <= '9' {
				p.pos = q
				for p.pos < len(p.src) && p.src[p.pos] >= '0' && p.src[p.pos] <= '9' {
					p.pos++
				}
			}
		}
		text := strings.ReplaceAll(p.src[start:p.pos], "_", "")
		v, err := strconv.ParseFloat(text, 64)
		if err != nil {
			p.tok = token{kind: tokOp, text: text}
			return
		}
		p.tok = token{kind: tokNum, num: v, text: text}
	case unicode.IsLetter(rune(c)):
		start := p.pos
		for p.pos < len(p.src) && (unicode.IsLetter(rune(p.src[p.pos])) || unicode.IsDigit(rune(p.src[p.pos]))) {
			p.pos++
		}
		p.tok = token{kind: tokIdent, text: strings.ToLower(p.src[start:p.pos])}
	default:
		p.pos++
		if c == '*' && p.pos < len(p.src) && p.src[p.pos] == '*' {
			p.pos++
			p.tok = token{kind: tokOp, text: "^"}
			return
		}
		p.tok = token{kind: tokOp, text: string(c)}
	}
}

func (p *calcParser) expr() (float64, error) {
	v, err := p.term()
	if err != nil {
		return 0, err
	}
	for p.tok.kind == tokOp && (p.tok.text == "+" || p.tok.text == "-") {
		op := p.tok.text
		p.next()
		r, err := p.term()
		if err != nil {
			return 0, err
		}
		if op == "+" {
			v += r
		} else {
			v -= r
		}
	}
	return v, nil
}

func (p *calcParser) term() (float64, error) {
	v, err := p.unary()
	if err != nil {
		return 0, err
	}
	for p.tok.kind == tokOp && (p.tok.text == "*" || p.tok.text == "/" || p.tok.text == "%") {
		op := p.tok.text
		p.next()
		r, err := p.unary()
		if err != nil {
			return 0, err
		}
		switch op {
		case "*":
			v *= r
		case "/":
			if r == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			v /= r
		case "%":
			v = math.Mod(v, r)
		}
	}
	return v, nil
}

func (p *calcParser) unary() (float64, error) {
	if p.tok.kind == tokOp && (p.tok.text == "-" || p.tok.text == "+") {
		neg := p.tok.text == "-"
		p.next()
		v, err := p.unary()
		if neg {
			v = -v
		}
		return v, err
	}
	return p.power()
}

func (p *calcParser) power() (float64, error) {
	base, err := p.postfix()
	if err != nil {
		return 0, err
	}
	if p.tok.kind == tokOp && p.tok.text == "^" {
		p.next()
		exp, err := p.unary() // right associative
		if err != nil {
			return 0, err
		}
		return math.Pow(base, exp), nil
	}
	return base, nil
}

func (p *calcParser) postfix() (float64, error) {
	v, err := p.primary()
	if err != nil {
		return 0, err
	}
	for p.tok.kind == tokOp && p.tok.text == "!" {
		p.next()
		if v < 0 || v != math.Trunc(v) || v > 170 {
			return 0, fmt.Errorf("factorial needs a whole number from 0 to 170")
		}
		r := 1.0
		for i := 2.0; i <= v; i++ {
			r *= i
		}
		v = r
	}
	return v, nil
}

func (p *calcParser) primary() (float64, error) {
	switch p.tok.kind {
	case tokNum:
		v := p.tok.num
		p.next()
		return v, nil
	case tokOp:
		if p.tok.text == "(" {
			p.next()
			v, err := p.expr()
			if err != nil {
				return 0, err
			}
			if p.tok.text != ")" {
				return 0, fmt.Errorf("missing )")
			}
			p.next()
			return v, nil
		}
	case tokIdent:
		name := p.tok.text
		p.next()
		switch name {
		case "pi":
			return math.Pi, nil
		case "e":
			return math.E, nil
		case "tau":
			return 2 * math.Pi, nil
		}
		if p.tok.text != "(" {
			return 0, fmt.Errorf("unknown name %q", name)
		}
		p.next()
		var args []float64
		for p.tok.text != ")" {
			v, err := p.expr()
			if err != nil {
				return 0, err
			}
			args = append(args, v)
			if p.tok.text == "," {
				p.next()
				continue
			}
			if p.tok.text != ")" {
				return 0, fmt.Errorf("expected , or ) in %s()", name)
			}
		}
		p.next()
		return callFunc(name, args)
	}
	if p.tok.kind == tokEOF {
		return 0, fmt.Errorf("unexpected end of expression")
	}
	return 0, fmt.Errorf("unexpected %q", p.tok.text)
}

func callFunc(name string, a []float64) (float64, error) {
	one := map[string]func(float64) float64{
		"sqrt": math.Sqrt, "abs": math.Abs, "ln": math.Log, "log": math.Log10, "log10": math.Log10,
		"log2": math.Log2, "exp": math.Exp, "sin": math.Sin, "cos": math.Cos, "tan": math.Tan,
		"asin": math.Asin, "acos": math.Acos, "atan": math.Atan, "floor": math.Floor,
		"ceil": math.Ceil, "round": math.Round, "cbrt": math.Cbrt, "trunc": math.Trunc,
	}
	if f, ok := one[name]; ok {
		if len(a) != 1 {
			return 0, fmt.Errorf("%s takes one argument", name)
		}
		return f(a[0]), nil
	}
	switch name {
	case "pow":
		if len(a) != 2 {
			return 0, fmt.Errorf("pow takes two arguments")
		}
		return math.Pow(a[0], a[1]), nil
	case "min", "max":
		if len(a) == 0 {
			return 0, fmt.Errorf("%s needs arguments", name)
		}
		v := a[0]
		for _, x := range a[1:] {
			if name == "min" {
				v = math.Min(v, x)
			} else {
				v = math.Max(v, x)
			}
		}
		return v, nil
	}
	return 0, fmt.Errorf("unknown function %s()", name)
}

// FormatNumber prints v without float noise.
func FormatNumber(v float64) string {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return fmt.Sprint(v)
	}
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'g', 12, 64)
}
