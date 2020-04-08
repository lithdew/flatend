package flatend

import (
	"errors"
	"fmt"
	"go.starlark.net/starlark"
)

func SQL(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return nil, nil
}

func GET(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) == 0 {
		return nil, errors.New("http_get: route path must be specified")
	}

	path, ok := starlark.AsString(args[0])
	if !ok {
		return nil, errors.New("http_get: route path must be a string")
	}

	fmt.Printf("Registering HTTP GET route: %q\n", path)

	return starlark.None, nil
}
