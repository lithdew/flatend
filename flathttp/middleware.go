package flathttp

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"
)

type Middleware struct {
	MinContentLength int64 // inclusive
	MaxContentLength int64 // inclusive

	Timeout time.Duration // max duration we are allowing a request to take

	Accepts []string // content types that are accepted
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.Timeout > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), m.Timeout)

		defer func() {
			cancel()

			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return
			}

			tmpl := `{"error": "request took too long", "max_timeout": "[[TIMEOUT]]"}`

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Content-Type-Options", "nosniff")

			w.WriteHeader(http.StatusGatewayTimeout)
			w.Write(T(tmpl, F{"TIMEOUT": m.Timeout.String()}))
		}()
	}

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
