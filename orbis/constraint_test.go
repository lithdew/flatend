package orbis

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"math"
	"strconv"
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

type rule struct {
	prec  int
	right bool
}

var rules = [...]rule{
	tokNegate: {prec: 4, right: true},

	tokGT:  {prec: 3, right: true},
	tokGTE: {prec: 3, right: true},
	tokLT:  {prec: 3, right: true},
	tokLTE: {prec: 3, right: true},

	tokPlus:  {prec: 2},
	tokMinus: {prec: 2},

	tokAND: {prec: 1},
	tokOR:  {prec: 1},
}

const eof = rune(0)

type constraintRule struct {
	is    int64
	ie    int64
	fs    float64
	fe    float64
	texts []string
}

func newConstraintRule() constraintRule {
	return constraintRule{
		is: math.MinInt64,
		ie: math.MaxInt64,
		fs: math.Inf(-1),
		fe: math.Inf(1),
	}
}

type constraint struct {
	rules []constraintRule
}

func newConstraint() constraint {
	return constraint{rules: []constraintRule{newConstraintRule()}}
}

func (c *constraint) split() {
	c.rules = append(c.rules, newConstraintRule())
}

func (c *constraint) updateIntMin(min int64) {
	i := len(c.rules) - 1
	if c.rules[i].is < min {
		c.rules[i].is = min
	}
}

func (c *constraint) updateIntMax(max int64) {
	i := len(c.rules) - 1
	if c.rules[i].ie > max {
		c.rules[i].ie = max
	}
}

func (c *constraint) updateFloatMin(min float64) {
	i := len(c.rules) - 1
	if c.rules[i].fs < min {
		c.rules[i].fs = min
	}
}

func (c *constraint) updateFloatMax(max float64) {
	i := len(c.rules) - 1
	if c.rules[i].fe > max {
		c.rules[i].fe = max
	}
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

func TestConstraint(t *testing.T) {
	q := `>=-50 & <= 400 | >=500 & <=600`

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

	//val := "12394"
	//r, _ := utf8.DecodeRuneInString(val)
	//
	//switch {
	//case isDecimalRune(r):
	//	if strings.ContainsRune(val, '.') {
	//		typ = tokFloat
	//	} else {
	//		typ = tokInt
	//	}
	//case r == '.':
	//	typ = tokFloat
	//default:
	//	typ = tokText
	//}

	res := newConstraint()

	ops := make([]token, 0, 64)
	vals := make([]token, 0, 64)

	eval := func(op token) {
		switch op.typ {
		case tokNegate:
			rhs := vals[len(vals)-1]

			switch rhs.typ {
			case tokInt, tokFloat:
				vals[len(vals)-1].neg = !rhs.neg
			default:
				panic(`unary '-' is not paired with int or float`)
			}
		case tokGT:
			rhs := vals[len(vals)-1]
			vals = vals[:len(vals)-1]

			switch rhs.typ {
			case tokInt:
				val, err := strconv.ParseInt(rhs.repr(q), 0, 64)
				if err != nil {
					panic("invalid int")
				}
				if rhs.neg {
					val = -val
				}
				res.updateIntMin(val + 1)
			case tokFloat:
				val, err := strconv.ParseFloat(rhs.repr(q), 64)
				if err != nil {
					panic("invalid float")
				}
				if rhs.neg {
					val = -val
				}
				res.updateFloatMin(val + 1)
			default:
				panic(`'>' is not paired with int or float`)
			}
		case tokGTE:
			rhs := vals[len(vals)-1]
			vals = vals[:len(vals)-1]

			switch rhs.typ {
			case tokInt:
				val, err := strconv.ParseInt(rhs.repr(q), 0, 64)
				if err != nil {
					panic("invalid int")
				}
				if rhs.neg {
					val = -val
				}
				res.updateIntMin(val)
			case tokFloat:
				val, err := strconv.ParseFloat(rhs.repr(q), 64)
				if err != nil {
					panic("invalid float")
				}
				if rhs.neg {
					val = -val
				}
				res.updateFloatMin(val)
			default:
				panic(`'>=' is not paired with int or float`)
			}
		case tokLT:
			rhs := vals[len(vals)-1]
			vals = vals[:len(vals)-1]

			switch rhs.typ {
			case tokInt:
				val, err := strconv.ParseInt(rhs.repr(q), 0, 64)
				if err != nil {
					panic("invalid int")
				}
				if rhs.neg {
					val = -val
				}
				res.updateIntMax(val - 1)
			case tokFloat:
				val, err := strconv.ParseFloat(rhs.repr(q), 64)
				if err != nil {
					panic("invalid float")
				}
				if rhs.neg {
					val = -val
				}
				res.updateFloatMax(val - 1)
			default:
				panic(`'<' is not paired with int or float`)
			}
		case tokLTE:
			rhs := vals[len(vals)-1]
			vals = vals[:len(vals)-1]

			switch rhs.typ {
			case tokInt:
				val, err := strconv.ParseInt(rhs.repr(q), 0, 64)
				if err != nil {
					panic("invalid int")
				}
				if rhs.neg {
					val = -val
				}
				res.updateIntMax(val)
			case tokFloat:
				val, err := strconv.ParseFloat(rhs.repr(q), 64)
				if err != nil {
					panic("invalid float")
				}
				if rhs.neg {
					val = -val
				}
				res.updateFloatMax(val)
			default:
				panic(`'<=' is not paired with int or float`)
			}
		case tokAND:
			fmt.Println("Hit &!")
			spew.Dump(ops, vals)
			fmt.Println()
		case tokOR:
			fmt.Println("Hit |!")
			spew.Dump(ops, vals)
			fmt.Println()

			res.split()
		}
	}

	current := lex()
	last := current

	for current.typ != tokEOF {
		fmt.Printf("%s\n", current.repr(q))

		switch current.typ {
		case tokText, tokInt, tokFloat:
			vals = append(vals, current)
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

				eval(op)
			}
			ops = append(ops, current)
		}

		spew.Dump(ops, vals)
		fmt.Println()

		last = current
		current = lex()
	}

	for len(ops) > 0 {
		op := ops[len(ops)-1]
		ops = ops[:len(ops)-1]

		if op.typ == tokBracketStart {
			panic("mismatched parenthesis")
		}

		eval(op)
	}

	spew.Dump(res)
}
