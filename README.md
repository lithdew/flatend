# flatend

[![MIT License](https://img.shields.io/apm/l/atomic-design-ui.svg?)](LICENSE)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lithdew/flatend)
[![Discord Chat](https://img.shields.io/discord/697002823123992617)](https://discord.gg/HZEbkeQ)

```go
package main

import (
	"github.com/lithdew/flatend"
	"strconv"
	"sync/atomic"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var counter uint64 = 0

func handleAllTodos(_ *flatend.Context) []byte {
	return strconv.AppendUint(nil, atomic.AddUint64(&counter, 1), 10)
}

func handleGetTodos(ctx *flatend.Context) []byte {
	return ctx.Body()
}

func main() {
	service := &flatend.Service{
		Addr: "127.0.0.1:9000",
		Services: map[string]flatend.Handler{
			"all_todos": handleAllTodos,
			"get_todos": handleGetTodos,
		},
	}
	check(service.Start())
}
```