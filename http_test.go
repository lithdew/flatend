package flatend

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseRoute(t *testing.T) {
	route := "GET /post/:id/la/rwar/gfdsgsdfg/"

	r, err := ParseRoute(route)
	require.NoError(t, err)

	require.EqualValues(t, "GET", r.Method)
	require.EqualValues(t, "/post/:id/la/rwar/gfdsgsdfg/", r.Path)

	r, err = ParseRoute("  /fail/")
	require.Error(t, err)
}
