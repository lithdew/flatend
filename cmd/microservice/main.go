package main

import (
	"github.com/lithdew/flatend"
	"os"
	"os/signal"
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
	node := &flatend.Node{
		Services: map[string]flatend.Handler{
			"all_todos": handleAllTodos,
			"get_todos": handleGetTodos,
		},
	}
	check(node.Start("0.0.0.0:9000"))

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	node.Shutdown()
}
