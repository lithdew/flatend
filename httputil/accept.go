package httputil

import (
	"net/http"
	"strings"
)

// AcceptSpec specifies one of potentially many media types that may be accepted by a client, weighed by a
// quality score denoted as a decimal in the range [0.0, 1.0].
type AcceptSpec struct {
	Value string
	Q     float64
}

// ParseAccept parses Accept* headers.
func ParseAccept(header http.Header, key string) (specs []AcceptSpec) {
	var spec AcceptSpec

	for _, s := range header[key] {
		for {
			spec.Value, s = expectTokenSlash(s)
			if spec.Value == "" {
				break
			}

			s = skipSpace(s)

			spec.Q = 1.0

			if len(s) > 0 && s[0] == ';' {
				s = skipSpace(s[1:])
				if !strings.HasPrefix(s, "q=") {
					break
				}
				spec.Q, s = expectQuality(s[2:])
				if spec.Q < 0.0 {
					break
				}
			}

			specs = append(specs, spec)

			s = skipSpace(s)
			if len(s) == 0 || s[0] != ',' {
				break
			}
			s = skipSpace(s[1:])
		}
	}
	return specs
}

func skipSpace(s string) (rest string) {
	i := 0
	for ; i < len(s); i++ {
		if octetTypes[s[i]]&isSpace == 0 {
			break
		}
	}
	return s[i:]
}

func expectTokenSlash(s string) (token, rest string) {
	i := 0
	for ; i < len(s); i++ {
		b := s[i]
		if (octetTypes[b]&isToken == 0) && b != '/' {
			break
		}
	}
	return s[:i], s[i:]
}

func expectQuality(s string) (q float64, rest string) {
	switch {
	case len(s) == 0:
		return -1, ""
	case s[0] == '0':
		q = 0
	case s[0] == '1':
		q = 1
	default:
		return -1, ""
	}
	s = s[1:]
	if len(s) == 0 || s[0] != '.' {
		return q, s
	}
	s = s[1:]
	i := 0
	n := 0
	d := 1
	for ; i < len(s); i++ {
		b := s[i]
		if b < '0' || b > '9' {
			break
		}
		n = n*10 + int(b) - '0'
		d *= 10
	}
	return q + float64(n)/float64(d), s[i:]
}
