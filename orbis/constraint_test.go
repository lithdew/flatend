package orbis

import (
	"fmt"
	"testing"
	"unicode/utf8"
)

type tokenType int

const (
	tokenGT tokenType = iota
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

const whitespace = uint64(1<<'\t' | 1<<'\n' | 1<<'\r' | 1<<' ')

func isWhitespace(r rune) bool {
	return whitespace&(1<<uint(r)) != 0
}

func lower(r rune) rune {
	return ('a' - 'A') | r
}

func isDigitRune(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHexRune(r rune) bool {
	return isDigitRune(r) || (r >= 'a' && r <= 'f')
}

func isBinRune(r rune) bool {
	return r >= '0' && r <= '7'
}

func TestConstraint(t *testing.T) {
	q := `"\377" | 0.123e4 & "yes"`

	bc := 0   // byte count
	cc := 0   // char count
	lcw := -1 // last char width

	next := func() rune {
		if bc == len(q) {
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

	tokens := make([]token, 0, len(q)/2)

	lexEscapeChar := func(quote rune) {
		r := next()

		switch r {
		case eof:
			panic("got eof while parsing escape literal")
		case quote, 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\':
			// ignore
		case 'x':
			for n := 0; n < 2; n++ {
				r := lower(next())
				if !isHexRune(r) || r == eof {
					panic("bad 16")
				}
			}
		case 'u':
			for n := 0; n < 4; n++ {
				r := lower(next())
				if !isHexRune(r) || r == eof {
					panic("bad 32")
				}
			}
		case 'U':
			for n := 0; n < 8; n++ {
				r := lower(next())
				if !isHexRune(r) || r == eof {
					panic("bad 64")
				}
			}
		default:
			if !isBinRune(r) || r == eof {
				panic("bad 8")
			}
			for n := 0; n < 2; n++ {
				r = next()
				if !isBinRune(r) || r == eof {
					panic("bad 8")
				}
			}
		}
	}

	lexNumber := func(r rune) (float bool) {
		var (
			separator bool
			digit     bool
			prefix    rune
		)

		float = r == '.'

		if r == '0' {
			prefix = lower(next())

			switch prefix {
			case 'x':
				for {
					r = lower(next())
					switch {
					case r == '_':
						separator = true
						continue
					case isHexRune(r):
						digit = true
						continue
					}
					break
				}
			case 'o':
				for {
					r = next()
					switch {
					case r == '_':
						separator = true
						continue
					case isBinRune(r):
						digit = true
						continue
					}
					break
				}
			case 'b':
				for {
					r = next()
					switch r {
					case '_':
						separator = true
						continue
					case '0', '1':
						digit = true
						continue
					}
					break
				}
			default:
				prefix, digit = '0', true

				for {
					switch {
					case r == '_':
						separator = true
						r = next()
						continue
					case isBinRune(r):
						r = next()
						continue
					}
					break
				}
			}
		} else {
			for {
				switch {
				case r == '_':
					separator = true
					r = next()
					continue
				case isDigitRune(r):
					digit = true
					r = next()
					continue
				}
				break
			}
		}

		// ^ stops at the first offending character

		if !float {
			float = r == '.'
		}

		if float {
			if prefix == 'o' || prefix == 'b' {
				panic("invalid radix point")
			}

			switch prefix {
			case 'x':
				for {
					r = lower(next())
					switch {
					case r == '_':
						separator = true
						continue
					case isHexRune(r):
						digit = true
						continue
					}
					break
				}
			case '0':
				for {
					r = next()
					switch {
					case r == '_':
						separator = true
						continue
					case isBinRune(r):
						continue
					}
					break
				}
			default:
				for {
					r = next()
					switch {
					case r == '_':
						separator = true
						continue
					case isDigitRune(r):
						digit = true
						continue
					}
					break
				}
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

			for {
				switch {
				case r == '_':
					r = next()
					separator = true
					continue
				case isDigitRune(r):
					r = next()
					digit = true
					continue
				}
				break
			}

			// ^ stops at the first offending character

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

	lexText := func(quote rune) {
		tokens = append(tokens, token{typ: tokenStringStart, ts: bc - 1, te: bc})

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
				tokens = append(tokens, token{typ: tokenText, ts: start, te: end})
			}
			tokens = append(tokens, token{typ: tokenStringEnd, ts: bc - 1, te: bc})
		} else {
			panic("unterminated")
		}
	}

	for {
		r := next()
		for isWhitespace(r) {
			r = next()
		}
		if r == eof {
			break
		}

		if isDigitRune(r) || r == '.' {
			s := bc - 1

			if lexNumber(r) {
				tokens = append(tokens, token{typ: tokenFloat, ts: s, te: bc})
			} else {
				tokens = append(tokens, token{typ: tokenInt, ts: s, te: bc})
			}

			continue
		}

		switch r {
		case '\'', '"':
			lexText(r)
		case '>':
			r = next()
			if r == '=' {
				tokens = append(tokens, token{typ: tokenGTE, ts: bc - 2, te: bc})
			} else {
				prev()
				tokens = append(tokens, token{typ: tokenGT, ts: bc - 1, te: bc})
			}
		case '<':
			r = next()
			if r == '=' {
				tokens = append(tokens, token{typ: tokenLTE, ts: bc - 2, te: bc})
			} else {
				prev()
				tokens = append(tokens, token{typ: tokenLT, ts: bc - 1, te: bc})
			}
		case '(':
			tokens = append(tokens, token{typ: tokenBracketStart, ts: bc - 1, te: bc})
		case ')':
			tokens = append(tokens, token{typ: tokenBracketEnd, ts: bc - 1, te: bc})
		case '&':
			tokens = append(tokens, token{typ: tokenAND, ts: bc - 1, te: bc})
		case '|':
			tokens = append(tokens, token{typ: tokenOR, ts: bc - 1, te: bc})
		}
	}

	for _, tok := range tokens {
		fmt.Printf("%s\n", tok.repr(q))
	}
}
