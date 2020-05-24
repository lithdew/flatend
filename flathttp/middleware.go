package flathttp

import (
	"net/http"
	"strconv"
)

type Middleware struct {
	MinContentLength int64 // inclusive
	MaxContentLength int64 // inclusive

	Accepts []string // content types that are accepted
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength <= m.MinContentLength || r.ContentLength >= m.MaxContentLength {
		tmpl := `{"error": "content length is not acceptable", "given": [[GIVEN]], "min": [[MIN]], "max": [[MAX]]}`

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusBadRequest)

		w.Write(
			T(tmpl, F{
				"GIVEN": strconv.FormatInt(r.ContentLength, 10),
				"MIN":   strconv.FormatInt(m.MinContentLength, 10),
				"MAX":   strconv.FormatInt(m.MaxContentLength, 10),
			}),
		)
		return
	}
}
