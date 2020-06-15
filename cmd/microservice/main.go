package main

import (
	"encoding/json"
	"github.com/lithdew/flatend"
	"io"
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

func handleAllTodos(ctx *flatend.Context) {
	ctx.Write(strconv.AppendUint(nil, atomic.AddUint64(&counter, 1), 10))
}

func handleGetTodos(ctx *flatend.Context) {
	buf, err := json.Marshal(ctx.Headers)
	if err != nil {
		return
	}
	ctx.Write(buf)
}

func pipe(ctx *flatend.Context) {
	io.Copy(ctx, ctx.Body)
}

func main() {
	node := &flatend.Node{
		Services: map[string]flatend.Handler{
			"all_todos": handleAllTodos,
			"get_todos": handleGetTodos,
			"pipe":      pipe,
		},
	}
	check(node.Start("0.0.0.0:9000"))

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	node.Shutdown()
}
