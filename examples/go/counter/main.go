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

func main() {
	counter := uint64(0)

	node := &flatend.Node{
		Services: map[string]flatend.Handler{
			"count": func(ctx *flatend.Context) {
				current := atomic.AddUint64(&counter, 1) - 1
				ctx.Write(strconv.AppendUint(nil, current, 10))
			},
		},
	}
	check(node.Start("127.0.0.1:9000"))

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	node.Shutdown()
}
