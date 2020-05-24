package flathttp

import (
	"context"
	"errors"
	"golang.org/x/time/rate"
	"math"
	"net/http"
	"strconv"
	"time"
)

type Middleware struct {
	MinContentLength int64 // inclusive
	MaxContentLength int64 // inclusive

	RateLimiter       *rate.Limiter // rate limiter (use methods to set max allowed per second, burst rate limit, etc)
	WaitUponRateLimit bool          // respond immediately if rate limit exceeded

	Timeout time.Duration // max duration we are allowing a request to take

	Accepts []string // content types that are accepted
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.Timeout > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), m.Timeout)
		r = r.WithContext(ctx)

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

	if m.RateLimiter != nil {
		now := time.Now()
		slot := m.RateLimiter.ReserveN(now, 1)
		deadline, exists := r.Context().Deadline()
		delay := slot.DelayFrom(now)

		switch {
		case !slot.OK():
			fallthrough
		case m.WaitUponRateLimit && delay == rate.InfDuration:
			fallthrough
		case m.WaitUponRateLimit && exists && deadline.After(now.Add(delay)):
			fallthrough
		case !m.WaitUponRateLimit && delay > 0:
			slot.CancelAt(now)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Content-Type-Options", "nosniff")

			if delay < math.MaxInt64 {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(delay.Seconds()), 10))
			}

			fields := F{
				"MAX":   strconv.FormatFloat(float64(m.RateLimiter.Limit()), 'g', -1, 64),
				"BURST": strconv.Itoa(m.RateLimiter.Burst()),
				"WAIT":  delay.String(),
			}

			var b []byte
			if delay < math.MaxInt64 {
				b = T(`{"error": "endpoint is too busy: wait a bit", "max_per_second": [[MAX]], "burst_rate": [[BURST]], "wait_for": "[[WAIT]]"}`, fields)
			} else {
				b = T(`{"error": "endpoint is too busy: try again later", "max_per_second": [[MAX]], "burst_rate": [[BURST]]}`, fields)
			}

			w.WriteHeader(http.StatusTooManyRequests)
			w.Write(b)
			return
		case m.WaitUponRateLimit && delay > 0:
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-r.Context().Done():
				timer.Stop()
				return
			}
			timer.Stop()
		}
	}

	if r.ContentLength > 0 && (r.ContentLength <= m.MinContentLength || r.ContentLength >= m.MaxContentLength) {
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
