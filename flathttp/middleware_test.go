package flathttp

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

//func TestMiddlewareOnServer(t *testing.T) {
//	http.ListenAndServe(":44444", &Middleware{
//		MinContentLength: -1,
//
//		RateLimiter:       rate.NewLimiter(1, 1),
//		WaitUponRateLimit: true,
//	})
//}

func TestContentLength(t *testing.T) {
	cases := []struct {
		given, min, max int64
		err             bool
	}{
		{given: 10, min: 10, max: 100, err: true},
		{given: 100, min: 10, max: 100, err: true},
		{given: 11, min: 10, max: 100},
		{given: 99, min: 10, max: 100},
	}

	for _, test := range cases {
		m := &Middleware{MinContentLength: test.min, MaxContentLength: test.max}

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", bytes.NewReader(make([]byte, test.given)))
		m.ServeHTTP(w, r)

		check := (test.err && w.Code == http.StatusBadRequest) || (!test.err && w.Code == http.StatusOK)
		require.True(t, check)
	}
}
