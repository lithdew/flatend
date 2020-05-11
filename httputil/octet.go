package httputil

import (
	"math"
	"strings"
)

// Octet types from RFC 2616.
var octetTypes [math.MaxUint8 + 1]octetType

type octetType byte

const (
	isToken octetType = 1 << iota
	isSpace
)

func init() {
	for c := 0; c < len(octetTypes); c++ {
		var t octetType
		isCtl := c <= 31 || c == 127
		isChar := 0 <= c && c <= 127
		isSeparator := strings.IndexRune(" \t\"(),/:;<=>?@[]\\{}", rune(c)) >= 0
		if strings.IndexRune(" \t\r\n", rune(c)) >= 0 {
			t |= isSpace
		}
		if isChar && !isCtl && !isSeparator {
			t |= isToken
		}
		octetTypes[c] = t
	}
}
