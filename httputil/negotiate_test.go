package httputil

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

var negotiateContentTypeTests = []struct {
	s            string
	offers       []string
	defaultOffer string
	expect       string
}{
	{"text/html, */*;q=0", []string{"x/y"}, "", ""},
	{"text/html, */*", []string{"x/y"}, "", "x/y"},
	{"text/html, image/png", []string{"text/html", "image/png"}, "", "text/html"},
	{"text/html, image/png", []string{"image/png", "text/html"}, "", "image/png"},
	{"text/html, image/png; q=0.5", []string{"image/png"}, "", "image/png"},
	{"text/html, image/png; q=0.5", []string{"text/html"}, "", "text/html"},
	{"text/html, image/png; q=0.5", []string{"foo/bar"}, "", ""},
	{"text/html, image/png; q=0.5", []string{"image/png", "text/html"}, "", "text/html"},
	{"text/html, image/png; q=0.5", []string{"text/html", "image/png"}, "", "text/html"},
	{"text/html;q=0.5, image/png", []string{"image/png"}, "", "image/png"},
	{"text/html;q=0.5, image/png", []string{"text/html"}, "", "text/html"},
	{"text/html;q=0.5, image/png", []string{"image/png", "text/html"}, "", "image/png"},
	{"text/html;q=0.5, image/png", []string{"text/html", "image/png"}, "", "image/png"},
	{"image/png, image/*;q=0.5", []string{"image/jpg", "image/png"}, "", "image/png"},
	{"image/png, image/*;q=0.5", []string{"image/jpg"}, "", "image/jpg"},
	{"image/png, image/*;q=0.5", []string{"image/jpg", "image/gif"}, "", "image/jpg"},
	{"image/png, image/*", []string{"image/jpg", "image/gif"}, "", "image/jpg"},
	{"image/png, image/*", []string{"image/gif", "image/jpg"}, "", "image/gif"},
	{"image/png, image/*", []string{"image/gif", "image/png"}, "", "image/png"},
	{"image/png, image/*", []string{"image/png", "image/gif"}, "", "image/png"},
}

func TestNegotiateContentType(t *testing.T) {
	for _, tt := range negotiateContentTypeTests {
		r := &http.Request{Header: http.Header{"Accept": {tt.s}}}
		require.EqualValues(t, tt.expect, NegotiateContentType(r, tt.offers, tt.defaultOffer))
	}
}
