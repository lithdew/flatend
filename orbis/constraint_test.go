package orbis

import (
	"fmt"
	"testing"
	"unicode/utf8"
)

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenGT
	tokenGTE
	tokenLT
	tokenLTE
	tokenAND
	tokenOR
	tokenText
	tokenInt
	tokenFloat
	tokenStringStart
	tokenStringEnd
	tokenBracketStart
	tokenBracketEnd
)

const eof = rune(0)

type token struct {
	typ    tokenType
	ts, te int // token start and end index
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
	q := `"\377" | 0.123e4 & "yes"`

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
			if r == '+' || r == '-' {
				r = next()
			}

			float = true
			digit = false

			skip(isDecimalRune)

			if !digit {
				panic("exponent has no digits")
			}
		} else if float && prefix == 'x' {
			panic("hexadecimal mantissa requires a 'p' exponent")
		}

		_ = separator

		prev()

		return float
	}

	lexText := func(quote rune) token {
		//tokens = append(tokens, token{typ: tokenStringStart, ts: bc - 1, te: bc})

		start, end := bc, -1

		for {
			switch next() {
			default:
				continue
			case '\\':
				lexEscapeChar(quote)
				continue
			case quote:
				end = bc - 1
			case eof, '\n':
			}
			break
		}

		if end != -1 {
			if start != end {
				return token{typ: tokenText, ts: start, te: end}
			}
			//tokens = append(tokens, token{typ: tokenStringEnd, ts: bc - 1, te: bc})
		}
		panic("unterminated")
	}

	lex := func() token {
		r := next()
		for isWhitespace(r) {
			r = next()
		}
		if r == eof {
			return token{typ: tokenEOF, ts: bc - 1, te: bc}
		}

		if isDecimalRune(r) || r == '.' {
			s := bc - 1

			if lexNumber(r) {
				return token{typ: tokenFloat, ts: s, te: bc}
			}
			return token{typ: tokenInt, ts: s, te: bc}
		}

		switch r {
		case '\'', '"':
			return lexText(r)
		case '>':
			r = next()
			if r == '=' {
				return token{typ: tokenGTE, ts: bc - 2, te: bc}
			} else {
				prev()
				return token{typ: tokenGT, ts: bc - 1, te: bc}
			}
		case '<':
			r = next()
			if r == '=' {
				return token{typ: tokenLTE, ts: bc - 2, te: bc}
			} else {
				prev()
				return token{typ: tokenLT, ts: bc - 1, te: bc}
			}
		case '(':
			return token{typ: tokenBracketStart, ts: bc - 1, te: bc}
		case ')':
			return token{typ: tokenBracketEnd, ts: bc - 1, te: bc}
		case '&':
			return token{typ: tokenAND, ts: bc - 1, te: bc}
		case '|':
			return token{typ: tokenOR, ts: bc - 1, te: bc}
		}
		panic(fmt.Sprintf("unknown rune %q", string(r)))
	}

	for tok := lex(); tok.typ != tokenEOF; tok = lex() {
		fmt.Printf("%s\n", tok.repr(q))
	}
}
