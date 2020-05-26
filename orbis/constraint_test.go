package orbis

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

type tokType int

const (
	tokEOF tokType = iota
	tokGT
	tokGTE
	tokLT
	tokLTE
	tokAND
	tokOR
	tokPlus
	tokMinus
	tokNegate
	tokText
	tokInt
	tokFloat
	tokBracketStart
	tokBracketEnd
)

var tokStr = [...]string{
	tokEOF:          "eof",
	tokGT:           ">",
	tokGTE:          ">=",
	tokLT:           "<",
	tokLTE:          "<=",
	tokAND:          "&",
	tokOR:           "|",
	tokPlus:         "+",
	tokMinus:        "-",
	tokNegate:       "-",
	tokText:         "text",
	tokInt:          "int",
	tokFloat:        "float",
	tokBracketStart: "(",
	tokBracketEnd:   ")",
}

func (t tokType) String() string {
	return tokStr[t]
}

const eof = rune(0)

var rules = [...]struct {
	prec  int
	right bool
}{
	tokNegate: {prec: 4, right: true},
	tokGT:     {prec: 4, right: true},
	tokGTE:    {prec: 4, right: true},
	tokLT:     {prec: 4, right: true},
	tokLTE:    {prec: 4, right: true},

	tokPlus:  {prec: 3},
	tokMinus: {prec: 3},

	tokAND: {prec: 2},
	tokOR:  {prec: 1},
}

type token struct {
	typ tokType // token type
	ts  int     // token start index
	te  int     // token end index
	neg bool    // is this token negated?
}

func (t token) repr(q string) string {
	return q[t.ts:t.te]
}

func lower(r rune) rune {
	return ('a' - 'A') | r
}

const whitespace = uint64(1<<'\t' | 1<<'\n' | 1<<'\r' | 1<<' ')

func isWhitespace(r rune) bool {
	return whitespace&(1<<uint(r)) != 0
}

func isBinRune(r rune) bool {
	return r == '0' || r == '1'
}

func isOctalRune(r rune) bool {
	return r >= '0' && r <= '7'
}

func isDecimalRune(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHexRune(r rune) bool {
	if isDecimalRune(r) {
		return true
	} else {
		r = lower(r)
	}
	return r >= 'a' && r <= 'f'
}

type nodeType int

const (
	nodeBool nodeType = iota
	nodeInt
	nodeFloat
	nodeText
)

type node struct {
	typ   nodeType
	bool  bool
	int   int64
	float float64
	text  string
}

func TestConstraint(t *testing.T) {
	q := `>=100 | "test"`
	val := "test"

	bc := 0   // byte count
	cc := 0   // char count
	lcw := -1 // last char width

	next := func() rune {
		if bc >= len(q) {
			if bc > len(q) {
				panic("went too far ahead")
			}
			return eof
		}
		r, cw := utf8.DecodeRuneInString(q[bc:])
		bc += cw
		lcw = cw
		cc++
		return r
	}

	prev := func() {
		if lcw < 0 {
			panic("went back too far")
		}
		bc -= lcw
		lcw = -1
		cc--
	}

	lexEscapeChar := func(quote rune) {
		r := next()

		skip := func(n int, pred func(rune) bool) {
			for n > 0 {
				r = next()
				if !pred(r) || r == eof {
					panic("bad")
				}
				n--
			}
		}

		switch r {
		case eof:
			panic("got eof while parsing escape literal")
		case quote, 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\':
			// ignore
		case 'x':
			skip(2, isHexRune)
		case 'u':
			skip(4, isHexRune)
		case 'U':
			skip(8, isHexRune)
		default:
			if !isOctalRune(r) || r == eof {
				panic("bad 8")
			}
			skip(2, isOctalRune)
		}
	}

	lexNumber := func(r rune) (float bool) {
		var (
			separator bool
			digit     bool
			prefix    rune
		)

		float = r == '.'

		skip := func(pred func(rune) bool) {
			for {
				switch {
				case r == '_':
					separator = true
					r = next()
					continue
				case pred(r):
					digit = true
					r = next()
					continue
				default:
					prev()
				case r == eof:
				}
				break
			}
		}

		if r == '0' {
			prefix = lower(next())

			switch prefix {
			case 'x':
				r = next()
				skip(isHexRune)
			case 'o':
				r = next()
				skip(isOctalRune)
			case 'b':
				r = next()
				skip(isBinRune)
			default:
				prefix, digit = '0', true
				skip(isOctalRune)
			}
		} else {
			skip(isDecimalRune)
		}

		if !float {
			float = r == '.'
		}

		if float {
			if prefix == 'o' || prefix == 'b' {
				panic("invalid radix point")
			}

			r = lower(next())
			r = lower(next())

			switch prefix {
			case 'x':
				skip(isHexRune)
			case '0':
				skip(isOctalRune)
			default:
				skip(isDecimalRune)
			}
		}

		if !digit {
			panic("number has no digits")
		}

		e := lower(r)

		if e == 'e' || e == 'p' {
			if e == 'e' && prefix != eof && prefix != '0' {
				panic(fmt.Sprintf("%q exponent requires decimal mantissa", r))
			}
			if e == 'p' && prefix != 'x' {
				panic(fmt.Sprintf("%q exponent requires hexadecimal mantissa", r))
			}

			r = next()
			r = next()
			if r == '+' || r == '-' {
				r = next()
			}

			float = true

			skip(isDecimalRune)

			if !digit {
				panic("exponent has no digits")
			}
		} else if float && prefix == 'x' {
			panic("hexadecimal mantissa requires a 'p' exponent")
		}

		_ = separator

		return float
	}

	lexText := func(quote rune) token {
		start := bc

		for {
			switch next() {
			case quote:
				return token{typ: tokText, ts: start, te: bc - 1}
			case '\\':
				lexEscapeChar(quote)
				continue
			case eof, '\n':
				panic("unterminated")
			default:
				continue
			}
		}
	}

	lex := func() token {
		r := next()
		for isWhitespace(r) {
			r = next()
		}
		if r == eof {
			return token{typ: tokEOF, ts: bc - 1, te: bc}
		}

		if isDecimalRune(r) || r == '.' {
			s := bc - 1

			if lexNumber(r) {
				return token{typ: tokFloat, ts: s, te: bc}
			}
			return token{typ: tokInt, ts: s, te: bc}
		}

		switch r {
		case '\'', '"':
			return lexText(r)
		case '>':
			r = next()
			if r == '=' {
				return token{typ: tokGTE, ts: bc - 2, te: bc}
			} else {
				prev()
				return token{typ: tokGT, ts: bc - 1, te: bc}
			}
		case '<':
			r = next()
			if r == '=' {
				return token{typ: tokLTE, ts: bc - 2, te: bc}
			} else {
				prev()
				return token{typ: tokLT, ts: bc - 1, te: bc}
			}
		case '+':
			return token{typ: tokPlus, ts: bc - 1, te: bc}
		case '-':
			return token{typ: tokMinus, ts: bc - 1, te: bc}
		case '(':
			return token{typ: tokBracketStart, ts: bc - 1, te: bc}
		case ')':
			return token{typ: tokBracketEnd, ts: bc - 1, te: bc}
		case '&':
			return token{typ: tokAND, ts: bc - 1, te: bc}
		case '|':
			return token{typ: tokOR, ts: bc - 1, te: bc}
		}
		panic(fmt.Sprintf("unknown rune %q", string(r)))
	}

	input := node{typ: nodeBool}

	r, _ := utf8.DecodeRuneInString(val)

	switch {
	case r == '-':
		fallthrough
	case isDecimalRune(r):
		if strings.ContainsRune(val, '.') {
			input.typ = nodeFloat
			val, err := strconv.ParseFloat(val, 64)
			if err != nil {
				panic(err)
			}
			input.float = val
		} else {
			input.typ = nodeInt
			val, err := strconv.ParseInt(val, 0, 64)
			if err != nil {
				panic(err)
			}
			input.int = val
		}
	case r == '.':
		input.typ = nodeFloat
		val, err := strconv.ParseFloat(val, 64)
		if err != nil {
			panic(err)
		}
		input.float = val
	default:
		input.typ = nodeText
		input.text = val
	}

	ops := make([]token, 0, 64)
	vals := make([]node, 0, 64)

	evalNode := func(val node) bool {
		switch val.typ {
		case nodeInt:
			return (input.typ == nodeInt && input.int == val.int) || (input.typ == nodeFloat && input.float == float64(val.int))
		case nodeFloat:
			return (input.typ == nodeFloat && input.float == val.float) || (input.typ == nodeInt && float64(input.int) == val.float)
		case nodeText:
			return input.typ == nodeText && input.text == val.text
		case nodeBool:
			return (input.typ == nodeBool && input.bool == val.bool) || val.bool
		}
		panic("should never happen")
	}

	evalOp := func(op token) {
		fmt.Printf("EVAL %q\n", op.repr(q))

		switch op.typ {
		case tokNegate:
			i := len(vals) - 1
			switch vals[i].typ {
			case nodeInt:
				vals[i].int = -vals[i].int
			case nodeFloat:
				vals[i].float = -vals[i].float
			default:
				panic(`unary '-' not paired with int or float`)
			}
		case tokGT:
			i := len(vals) - 1
			switch input.typ {
			case nodeInt:
				switch vals[i].typ {
				case nodeInt:
					vals[i] = node{typ: nodeBool, bool: input.int > vals[i].int}
				case nodeFloat:
					vals[i] = node{typ: nodeBool, bool: float64(input.int) > vals[i].float}
				default:
					panic(`'>' not paired with int or float`)
				}
			case nodeFloat:
				switch vals[i].typ {
				case nodeInt:
					vals[i] = node{typ: nodeBool, bool: input.float > float64(vals[i].int)}
				case nodeFloat:
					vals[i] = node{typ: nodeBool, bool: input.float > vals[i].float}
				default:
					panic(`'>' not paired with int or float`)
				}
			default:
				vals[i] = node{typ: nodeBool, bool: false}
			}
		case tokLT:
			i := len(vals) - 1
			switch input.typ {
			case nodeInt:
				switch vals[i].typ {
				case nodeInt:
					vals[i] = node{typ: nodeBool, bool: input.int < vals[i].int}
				case nodeFloat:
					vals[i] = node{typ: nodeBool, bool: float64(input.int) < vals[i].float}
				default:
					panic(`'<' not paired with int or float`)
				}
			case nodeFloat:
				switch vals[i].typ {
				case nodeInt:
					vals[i] = node{typ: nodeBool, bool: input.float < float64(vals[i].int)}
				case nodeFloat:
					vals[i] = node{typ: nodeBool, bool: input.float < vals[i].float}
				default:
					panic(`'<' not paired with int or float`)
				}
			default:
				vals[i] = node{typ: nodeBool, bool: false}
			}
		case tokGTE:
			i := len(vals) - 1
			switch input.typ {
			case nodeInt:
				switch vals[i].typ {
				case nodeInt:
					vals[i] = node{typ: nodeBool, bool: input.int >= vals[i].int}
				case nodeFloat:
					vals[i] = node{typ: nodeBool, bool: float64(input.int) >= vals[i].float}
				default:
					panic(`'>=' not paired with int or float`)
				}
			case nodeFloat:
				switch vals[i].typ {
				case nodeInt:
					vals[i] = node{typ: nodeBool, bool: input.float >= float64(vals[i].int)}
				case nodeFloat:
					vals[i] = node{typ: nodeBool, bool: input.float >= vals[i].float}
				default:
					panic(`'>=' not paired with int or float`)
				}
			default:
				vals[i] = node{typ: nodeBool, bool: false}
			}
		case tokLTE:
			i := len(vals) - 1
			switch input.typ {
			case nodeInt:
				switch vals[i].typ {
				case nodeInt:
					vals[i] = node{typ: nodeBool, bool: input.int <= vals[i].int}
				case nodeFloat:
					vals[i] = node{typ: nodeBool, bool: float64(input.int) <= vals[i].float}
				default:
					panic(`'<=' not paired with int or float`)
				}
			case nodeFloat:
				switch vals[i].typ {
				case nodeInt:
					vals[i] = node{typ: nodeBool, bool: input.float <= float64(vals[i].int)}
				case nodeFloat:
					vals[i] = node{typ: nodeBool, bool: input.float <= vals[i].float}
				default:
					panic(`'<=' not paired with int or float`)
				}
			default:
				vals[i] = node{typ: nodeBool, bool: false}
			}
		case tokAND:
			l := len(vals) - 2
			r := len(vals) - 1

			vals[l] = node{typ: nodeBool, bool: evalNode(vals[l]) && evalNode(vals[r])}
			vals = vals[:r]
		case tokOR:
			l := len(vals) - 2
			r := len(vals) - 1

			vals[l] = node{typ: nodeBool, bool: evalNode(vals[l]) || evalNode(vals[r])}
			vals = vals[:r]
		}
	}

	current := lex()
	last := current

	for current.typ != tokEOF {
		switch current.typ {
		case tokInt:
			val, err := strconv.ParseInt(current.repr(q), 0, 64)
			if err != nil {
				panic(err)
			}
			vals = append(vals, node{typ: nodeInt, int: val})
		case tokFloat:
			val, err := strconv.ParseFloat(current.repr(q), 64)
			if err != nil {
				panic(err)
			}
			vals = append(vals, node{typ: nodeFloat, float: val})
		case tokText:
			vals = append(vals, node{typ: nodeText, text: current.repr(q)})
		case tokBracketStart:
			ops = append(ops, current)
		case tokGT, tokGTE, tokLT, tokLTE, tokAND, tokOR, tokPlus, tokMinus:
			if current.typ == tokMinus && last.typ != tokInt && last.typ != tokFloat && last.typ != tokText {
				current.typ = tokNegate
			}

			for len(ops) > 0 {
				op := ops[len(ops)-1]

				o1 := rules[current.typ]
				o2 := rules[op.typ]

				if op.typ == tokBracketStart || o1.prec > o2.prec || o1.prec == o2.prec && o1.right {
					break
				}

				ops = ops[:len(ops)-1]

				evalOp(op)
			}
			ops = append(ops, current)
		}

		last = current
		current = lex()
	}

	for len(ops) > 0 {
		op := ops[len(ops)-1]
		ops = ops[:len(ops)-1]

		if op.typ == tokBracketStart {
			panic("mismatched parenthesis")
		}

		evalOp(op)
	}

	spew.Dump(vals)
}
